package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/parser"
)

func TestNormalizeCheckOption(t *testing.T) {
	parserOption := parser.Option{SkelIn: "./demo"}

	if err := normalizeParserOption(&parserOption); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(parserOption.SkelIn) {
		t.Fatalf("expected absolute skel-in, got %q", parserOption.SkelIn)
	}
}

func TestNormalizeCheckOptionRequiresInput(t *testing.T) {
	parserOption := parser.Option{}
	expectOptionError(t, normalizeParserOption(&parserOption), "missing flag skel-in")
}

func expectOptionError(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}
