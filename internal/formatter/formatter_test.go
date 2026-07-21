package formatter

import "testing"

func TestSource(t *testing.T) {
	source := []byte("  domain demo.user  \r\n\r\n\r\nservice UserService {\r\nmethod ping {\r\ninput {\r\nvalue: string   \r\n}\r\n}\r\n}\r\n")
	want := "domain demo.user\n\nservice UserService {\n    method ping {\n        input {\n            value: string\n        }\n    }\n}\n"
	got := string(Source(source))
	if got != want {
		t.Fatalf("unexpected formatted source:\n%s\nwant:\n%s", got, want)
	}
	if second := string(Source([]byte(got))); second != got {
		t.Fatalf("format is not idempotent:\n%s", second)
	}
}

func TestSourcePreservesCommentsAndStrings(t *testing.T) {
	source := []byte(`domain demo.user

/* comment { }
   keep */
@desc("""
  keep { content }
""")
service UserService {
for ClientActor
method ping {}
}
`)
	want := `domain demo.user

/* comment { }
   keep */
@desc("""
  keep { content }
""")
service UserService {
    for ClientActor
    method ping {}
}
`
	if got := string(Source(source)); got != want {
		t.Fatalf("unexpected formatted source:\n%s\nwant:\n%s", got, want)
	}
}
