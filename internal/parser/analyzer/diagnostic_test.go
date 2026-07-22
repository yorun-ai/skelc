package analyzer

import (
	"testing"

	"go.yorun.ai/skelc/internal/parser/grammar"
)

func TestDiagnosticReporterDoesNotUsePanicControlFlow(t *testing.T) {
	reporter := newDiagnosticReporter()

	if reporter.check(false, "invalid value") {
		t.Fatal("expected failed check")
	}
	if len(reporter.result()) != 1 {
		t.Fatalf("expected one diagnostic, got %d", len(reporter.result()))
	}
}

func TestAnalyzeReturnsErrorsWithoutPanicking(t *testing.T) {
	_, diagnostics := Analyze(&grammar.SkelContent{
		Domain: domainContent("demo"),
		Entries: []*grammar.SkelEntry{
			{Data: &grammar.Data{Name: ident("User"), Members: []*grammar.DataMember{
				{Name: ident("first"), Type: refGrammarType("MissingFirst")},
				{Name: ident("second"), Type: refGrammarType("MissingSecond")},
			}}},
		},
	}, nil)

	if len(diagnostics) != 2 {
		t.Fatalf("expected two diagnostics, got %d: %v", len(diagnostics), diagnostics)
	}
}
