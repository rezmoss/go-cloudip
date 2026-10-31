package source

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"math"
)

// MaxDecompressedSize caps gzip inflation of untrusted input.
const MaxDecompressedSize = 256 << 20

// ParseDatabase parses MessagePack data into a Database struct.
// It automatically detects and handles gzip compression.
func ParseDatabase(data []byte) (*Database, error) {
	decompressed, err := Decompress(data)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	db, err := decodeDatabase(decompressed)
	if err != nil {
		return nil, fmt.Errorf("msgpack decode: %w", err)
	}

	if len(db.Providers) == 0 {
		return nil, fmt.Errorf("database has no providers")
	}
	if len(db.Ranges) == 0 {
		return nil, fmt.Errorf("database has no ranges")
	}

	return db, nil
}

// Decompress decompresses data if it's gzip compressed.
// Returns the original data if not compressed.
func Decompress(data []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		return data, nil
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader creation failed: %w", err)
	}
	defer reader.Close()

	// ~9x ratio; prealloc avoids regrowth
	buf := bytes.NewBuffer(make([]byte, 0, 10*len(data)))
	n, err := buf.ReadFrom(io.LimitReader(reader, MaxDecompressedSize+1))
	if err != nil {
		return nil, fmt.Errorf("gzip decompression failed: %w", err)
	}
	if n > MaxDecompressedSize {
		return nil, fmt.Errorf("gzip data exceeds %d bytes", MaxDecompressedSize)
	}
	return buf.Bytes(), nil
}

// decodeDatabase: direct msgpack decode, no reflection. One alloc for all
// CIDRs, region/service interned. Unknown keys skipped for fwd compat.
func decodeDatabase(data []byte) (*Database, error) {
	d := &decoder{buf: data, text: string(data), intern: map[string]string{}}
	db := &Database{}
	n, err := d.mapLen()
	if err != nil {
		return nil, err
	}
	for range n {
		key, err := d.bytes()
		if err != nil {
			return nil, err
		}
		switch string(key) {
		case "version":
			db.Version, err = d.str()
		case "build_time":
			db.BuildTime, err = d.int()
		case "providers":
			db.Providers, err = d.strings()
		case "ranges":
			db.Ranges, err = d.ranges()
		default:
			err = d.skip(0)
		}
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", key, err)
		}
	}
	return db, nil
}

var errShort = errors.New("unexpected end of data")

type decoder struct {
	buf    []byte
	text   string // buf as string; CIDRs are substrings
	pos    int
	intern map[string]string
}

func (d *decoder) ranges() ([]Range, error) {
	n, err := d.arrayLen()
	if err != nil {
		return nil, err
	}
	if n > len(d.buf)-d.pos { // each range >= 1 byte
		return nil, errShort
	}
	out := make([]Range, n)
	for i := range out {
		r := &out[i]
		fields, err := d.mapLen()
		if err != nil {
			return nil, err
		}
		for range fields {
			key, err := d.bytes()
			if err != nil {
				return nil, err
			}
			switch string(key) {
			case "cidr":
				var start, end int
				if start, end, err = d.strSpan(); err == nil {
					r.CIDR = d.text[start:end]
				}
			case "p":
				var p int64
				p, err = d.int()
				if p < math.MinInt32 || p > math.MaxInt32 {
					p = -1
				}
				r.Provider = int(p)
			case "r":
				r.Region, err = d.internedStr()
			case "s":
				r.Service, err = d.internedStr()
			default:
				err = d.skip(0)
			}
			if err != nil {
				return nil, fmt.Errorf("range %d: %w", i, err)
			}
		}
	}
	return out, nil
}

func (d *decoder) strings() ([]string, error) {
	n, err := d.arrayLen()
	if err != nil {
		return nil, err
	}
	if n > len(d.buf)-d.pos {
		return nil, errShort
	}
	out := make([]string, n)
	for i := range out {
		if out[i], err = d.str(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (d *decoder) byte() (byte, error) {
	if d.pos >= len(d.buf) {
		return 0, errShort
	}
	c := d.buf[d.pos]
	d.pos++
	return c, nil
}

func (d *decoder) uint(size int) (uint64, error) {
	if len(d.buf)-d.pos < size {
		return 0, errShort
	}
	var v uint64
	for _, c := range d.buf[d.pos : d.pos+size] {
		v = v<<8 | uint64(c)
	}
	d.pos += size
	return v, nil
}

// container: map/array header. fix = fix tag, c16 = 16-bit, c16+1 = 32-bit.
func (d *decoder) container(what string, fix, c16 byte) (int, error) {
	c, err := d.byte()
	if err != nil {
		return 0, err
	}
	switch {
	case c&0xf0 == fix:
		return int(c & 0x0f), nil
	case c == c16:
		return d.lenN(2, 0)
	case c == c16+1:
		return d.lenN(4, 0)
	case c == 0xc0:
		return 0, nil
	}
	return 0, fmt.Errorf("expected %s, got 0x%02x", what, c)
}

func (d *decoder) mapLen() (int, error)   { return d.container("map", 0x80, 0xde) }
func (d *decoder) arrayLen() (int, error) { return d.container("array", 0x90, 0xdc) }

// strSpan: bounds of next str; nil = empty
func (d *decoder) strSpan() (int, int, error) {
	c, err := d.byte()
	if err != nil {
		return 0, 0, err
	}
	if c == 0xc0 {
		return d.pos, d.pos, nil
	}
	var n int
	switch c {
	case 0xd9, 0xc4: // str8, bin8 (lib accepts bin as str)
		n, err = d.lenN(1, 0)
	case 0xda, 0xc5:
		n, err = d.lenN(2, 0)
	case 0xdb, 0xc6:
		n, err = d.lenN(4, 0)
	default:
		if c&0xe0 != 0xa0 {
			return 0, 0, fmt.Errorf("expected string, got 0x%02x", c)
		}
		n = int(c & 0x1f)
	}
	if err != nil {
		return 0, 0, err
	}
	if n > len(d.buf)-d.pos {
		return 0, 0, errShort
	}
	d.pos += n
	return d.pos - n, d.pos, nil
}

func (d *decoder) bytes() ([]byte, error) {
	start, end, err := d.strSpan()
	return d.buf[start:end], err
}

func (d *decoder) str() (string, error) {
	start, end, err := d.strSpan()
	return string(d.buf[start:end]), err
}

func (d *decoder) internedStr() (string, error) {
	b, err := d.bytes()
	if err != nil || len(b) == 0 {
		return "", err
	}
	if s, ok := d.intern[string(b)]; ok {
		return s, nil
	}
	s := string(b)
	d.intern[s] = s
	return s, nil
}

func (d *decoder) int() (int64, error) {
	c, err := d.byte()
	if err != nil {
		return 0, err
	}
	switch {
	case c <= 0x7f:
		return int64(c), nil
	case c >= 0xe0:
		return int64(int8(c)), nil
	case c >= 0xcc && c <= 0xcf: // uint8..uint64
		v, err := d.uint(1 << (c - 0xcc))
		if v > math.MaxInt64 {
			return 0, errors.New("integer overflow")
		}
		return int64(v), err
	case c >= 0xd0 && c <= 0xd3: // int8..int64
		size := 1 << (c - 0xd0)
		v, err := d.uint(size)
		shift := 64 - 8*size
		return int64(v<<shift) >> shift, err
	case c == 0xc0:
		return 0, nil
	}
	return 0, fmt.Errorf("expected integer, got 0x%02x", c)
}

// skip: consume one value
func (d *decoder) skip(depth int) error {
	if depth > 32 {
		return errors.New("nesting too deep")
	}
	c, err := d.byte()
	if err != nil {
		return err
	}
	var n, elems int
	switch {
	case c <= 0x7f || c >= 0xe0 || c == 0xc0 || c == 0xc2 || c == 0xc3:
		return nil
	case c&0xe0 == 0xa0:
		n = int(c & 0x1f)
	case c&0xf0 == 0x90:
		elems = int(c & 0x0f)
	case c&0xf0 == 0x80:
		elems = 2 * int(c&0x0f)
	case c == 0xcc || c == 0xd0:
		n = 1
	case c == 0xcd || c == 0xd1:
		n = 2
	case c == 0xce || c == 0xd2 || c == 0xca:
		n = 4
	case c == 0xcf || c == 0xd3 || c == 0xcb:
		n = 8
	case c >= 0xd4 && c <= 0xd8: // fixext 1..16 + type byte
		n = 1<<(c-0xd4) + 1
	case c == 0xd9 || c == 0xc4:
		n, err = d.lenN(1, 0)
	case c == 0xda || c == 0xc5:
		n, err = d.lenN(2, 0)
	case c == 0xdb || c == 0xc6:
		n, err = d.lenN(4, 0)
	case c == 0xc7:
		n, err = d.lenN(1, 1)
	case c == 0xc8:
		n, err = d.lenN(2, 1)
	case c == 0xc9:
		n, err = d.lenN(4, 1)
	case c == 0xdc:
		elems, err = d.lenN(2, 0)
	case c == 0xdd:
		elems, err = d.lenN(4, 0)
	case c == 0xde:
		elems, err = d.lenN(2, 0)
		elems *= 2
	case c == 0xdf:
		elems, err = d.lenN(4, 0)
		elems *= 2
	default:
		return fmt.Errorf("invalid type 0x%02x", c)
	}
	if err != nil {
		return err
	}
	if n > len(d.buf)-d.pos {
		return errShort
	}
	d.pos += n
	for range elems {
		if err := d.skip(depth + 1); err != nil {
			return err
		}
	}
	return nil
}

func (d *decoder) lenN(size, extra int) (int, error) {
	v, err := d.uint(size)
	return int(v) + extra, err
}
