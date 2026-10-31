package source

import (
	"reflect"
	"testing"

	"github.com/vmihailenco/msgpack/v5"
)

// custom decoder must match vmihailenco/msgpack (used by cloudip-db)
func TestDecodeMatchesMsgpack(t *testing.T) {
	data, err := GetEmbeddedData()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := Decompress(data)
	if err != nil {
		t.Fatal(err)
	}
	var want Database
	if err := msgpack.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	got, err := ParseDatabase(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("decoded database differs: version %q/%q, %d/%d ranges", got.Version, want.Version, len(got.Ranges), len(want.Ranges))
	}
}

func TestDecodeVariants(t *testing.T) {
	type wide struct {
		Extra     map[string]any   `msgpack:"extra"`
		Version   string           `msgpack:"version"`
		BuildTime int64            `msgpack:"build_time"`
		Providers []string         `msgpack:"providers"`
		Ranges    []map[string]any `msgpack:"ranges"`
		Tail      []any            `msgpack:"tail"`
	}
	in := wide{
		Extra:     map[string]any{"a": []any{1.5, true, nil, "x", []byte{1, 2}, int64(-1 << 40)}},
		Version:   "v",
		BuildTime: 1 << 40,
		Providers: []string{"aws", "gcp"},
		Ranges: []map[string]any{
			{"cidr": "1.2.3.0/24", "p": 1, "r": "us", "s": "EC2", "new": map[string]any{"k": 1}},
			{"cidr": "::/0", "p": int64(0), "r": nil},
		},
		Tail: []any{float32(1), uint64(1 << 63)},
	}
	b, err := msgpack.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	db, err := ParseDatabase(b)
	if err != nil {
		t.Fatal(err)
	}
	want := &Database{Version: "v", BuildTime: 1 << 40, Providers: []string{"aws", "gcp"}, Ranges: []Range{
		{CIDR: "1.2.3.0/24", Provider: 1, Region: "us", Service: "EC2"},
		{CIDR: "::/0", Provider: 0},
	}}
	if !reflect.DeepEqual(db, want) {
		t.Fatalf("got %+v", db)
	}
}

func FuzzParseDatabase(f *testing.F) {
	b, _ := msgpack.Marshal(Database{Version: "v", Providers: []string{"a"}, Ranges: []Range{{CIDR: "1.0.0.0/8"}}})
	f.Add(b)
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseDatabase(data) // no panic/hang
	})
}
