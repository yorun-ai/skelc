package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/compiler"
	optionvalidation "go.yorun.ai/skelc/internal/option"
)

func TestNormalizeCheckOption(t *testing.T) {
	compilerOption := compiler.Option{SkelIn: "./demo"}

	if err := normalizeCompilerOption(&compilerOption); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(compilerOption.SkelIn) {
		t.Fatalf("expected absolute skel-in, got %q", compilerOption.SkelIn)
	}
}

func TestNormalizeCheckOptionRequiresInput(t *testing.T) {
	compilerOption := compiler.Option{}
	expectOptionError(t, normalizeCompilerOption(&compilerOption), "missing flag skel-in")
}

func TestFormatGenerationErrorUsesTypedValidationContract(t *testing.T) {
	err := fmt.Errorf("normalize options: %w", optionvalidation.NewValidationError(
		optionvalidation.FieldGoModule,
		optionvalidation.RuleRequiresModule,
		"API message",
	))
	formatted := formatGenerationError(err)
	if formatted.Error() != "flag go-module requires go-module output" {
		t.Fatalf("unexpected CLI validation message: %v", formatted)
	}
}

func expectOptionError(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}
