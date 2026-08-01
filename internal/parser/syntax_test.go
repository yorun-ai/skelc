package parser

import (
	"strings"
	"testing"
)

func TestValidateSource(t *testing.T) {
	if err := ValidateSource("/tmp/demo.skel", []byte("domain demo.user\n")); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSourceReturnsErrorForInvalidSyntax(t *testing.T) {
	err := ValidateSource("/tmp/demo.skel", []byte("domain {\n"))
	if err == nil || !strings.Contains(err.Error(), "parse /tmp/demo.skel failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSourceFragmentPreservesOffsets(t *testing.T) {
	result, err := ParseSourceFragment("demo.skel", []byte("data User { id: string }"), 4, 32)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content == nil || len(result.Content.Entries) != 1 {
		t.Fatalf("unexpected parsed content: %+v", result.Content)
	}
	position := result.Content.Entries[0].Data.Name.Pos
	if position.Line != 4 || position.Offset != 37 {
		t.Fatalf("unexpected fragment position: %+v", position)
	}
}
