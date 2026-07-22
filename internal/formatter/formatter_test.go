package formatter

import (
	"os"
	"path/filepath"
	"testing"

	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

func TestSourceGolden(t *testing.T) {
	input := readTestFile(t, "complete.input.skel")
	want := readTestFile(t, "complete.golden.skel")

	if _, err := parser.ParseSource("complete.input.skel", input); err != nil {
		t.Fatalf("input fixture does not parse: %v", err)
	}
	got := Source(input)
	if string(got) != string(want) {
		t.Fatalf("unexpected formatted source:\n%s\nwant:\n%s", got, want)
	}
	if _, err := parser.ParseSource("complete.golden.skel", got); err != nil {
		t.Fatalf("formatted fixture does not parse: %v", err)
	}
	if second := Source(got); string(second) != string(got) {
		t.Fatalf("format is not idempotent:\n%s", second)
	}
}

func TestFormatterIsIdempotentAroundUnmatchedClosingBrace(t *testing.T) {
	first := Source([]byte("data}0"))
	second := Source(first)
	if string(first) != string(second) {
		t.Fatalf("formatter is not idempotent: first=%q second=%q", first, second)
	}
}

func TestFormatterIsIdempotentAroundMismatchedParenAndBrace(t *testing.T) {
	first := Source([]byte("//00\n00000(}"))
	second := Source(first)
	if string(first) != string(second) {
		t.Fatalf("formatter is not idempotent: first=%q second=%q", first, second)
	}
}

func TestFormatterIsIdempotentAroundInlineTripleString(t *testing.T) {
	for _, source := range [][]byte{
		[]byte("0\"\"\"\n  \"\"\""),
		[]byte("{\"\"\"\r \r\"\"\""),
	} {
		first := Source(source)
		second := Source(first)
		if string(first) != string(second) {
			t.Errorf("formatter is not idempotent: first=%q second=%q", first, second)
		}
	}
}

func TestSourcePreservesCommentsAndStrings(t *testing.T) {
	source := []byte("domain demo.user\n\n/* comment { }\n   keep */\n@desc(\"\"\"\n  keep { content }\n    nested\n\"\"\") // inline\nservice UserService {\nmethod ping {}\n}\n")
	want := "domain demo.user\n\n/* comment { }\n   keep */\n@desc(\"\"\"\nkeep { content }\n  nested\n\"\"\") // inline\nservice UserService {\n    method ping {}\n}\n"

	got := Source(source)
	if string(got) != want {
		t.Fatalf("unexpected formatted source:\n%s\nwant:\n%s", got, want)
	}
	before := descriptionValue(t, source)
	after := descriptionValue(t, got)
	if before != after {
		t.Fatalf("format changed triple-string value: before=%q after=%q", before, after)
	}
}

func descriptionValue(t *testing.T, source []byte) string {
	t.Helper()
	content, err := parser.ParseSource("description.skel", source)
	if err != nil {
		t.Fatal(err)
	}
	raw := content.Entries[0].Service.Decorators[0].Value.Raw
	description, err := grammar.UnquoteDescriptionString(raw)
	if err != nil {
		t.Fatal(err)
	}
	return description
}

func TestSourcePreservesSemanticHash(t *testing.T) {
	source := []byte(`domain demo.user

pub actor ClientActor {
via client {}
}

pub data User {
id:uuid
}

pub service UserService {
for ClientActor via client
method get {
input {
id:uuid
}
output User?
}
}
`)
	formatted := Source(source)
	before := parseDomainHash(t, "before.skel", source)
	after := parseDomainHash(t, "after.skel", formatted)
	if before != after {
		t.Fatalf("format changed semantic hash: before=%s after=%s", before, after)
	}
}

func TestSourceNormalizesEmptyAndInvalidInput(t *testing.T) {
	if got := Source([]byte(" \r\n\t")); len(got) != 0 {
		t.Fatalf("expected empty output, got %q", got)
	}
	if got := string(Source([]byte("invalid !  \r\n"))); got != "invalid !\n" {
		t.Fatalf("unexpected invalid-source fallback: %q", got)
	}
}

func readTestFile(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func parseDomainHash(t *testing.T, name string, source []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, source, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := parser.Parse(parser.Option{SkelIn: path})
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return result.Domain.Hash()
}
