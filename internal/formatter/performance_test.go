package formatter

import "testing"

var formatterBenchmarkSource = []byte(`domain benchmark.demo
@desc("User payload")
pub data User<T> {
    id: uuid
    values: list<map<string, T?>>
}
`)

func BenchmarkSource(b *testing.B) {
	for range b.N {
		Source(formatterBenchmarkSource)
	}
}

func FuzzSourceIdempotent(f *testing.F) {
	f.Add(formatterBenchmarkSource)
	f.Add([]byte("domain fuzz\r\ndata User{id:string}\r\n"))
	f.Add([]byte("// comment\n@desc(\"value\")\ndata User { value: list<string?> }"))
	f.Add([]byte("0/*\n  */"))
	f.Add([]byte("{\n0/*\n  */"))
	f.Add([]byte("0\"\"\"\n  \"\"\""))
	f.Fuzz(func(t *testing.T, source []byte) {
		first := Source(source)
		second := Source(first)
		if string(first) != string(second) {
			t.Fatalf("formatter is not idempotent: first=%q second=%q", first, second)
		}
	})
}
