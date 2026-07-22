package parser

import (
	"strings"
	"testing"
)

func TestParseSourceRecoveringCollectsIndependentSyntaxErrors(t *testing.T) {
	source := []byte(`domain demo
data User {
    first string
    second:
    third: string
}
data Order {
    id string
}
`)
	content, diagnostics := ParseSourceRecovering("/workspace/domain.skel", source)
	if content == nil {
		t.Fatal("expected recovered syntax tree")
	}
	if len(diagnostics) != 3 {
		t.Fatalf("expected three syntax diagnostics, got %d: %v", len(diagnostics), diagnostics)
	}
	if len(content.Entries) != 2 {
		t.Fatalf("expected both top-level declarations to survive recovery, got %d", len(content.Entries))
	}
	if got := content.Entries[0].Data.Members; len(got) != 1 || got[0].Name.Value != "third" {
		t.Fatalf("unexpected recovered members: %+v", got)
	}
}

func TestWorkspaceReportsRecoveredSyntaxAndSemanticDiagnostics(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{{
		Path: "/workspace/domain.skel",
		Content: []byte(`domain demo
data User {
    id string
}
data User {}
`),
	}})
	if len(diagnostics) != 2 {
		t.Fatalf("expected syntax and semantic diagnostics, got %d: %v", len(diagnostics), diagnostics)
	}
	if !strings.HasPrefix(diagnostics[0].Code, "syntax.") || diagnostics[1].Code != DiagnosticCodeSemanticDuplicate {
		t.Fatalf("unexpected diagnostic codes: %s, %s", diagnostics[0].Code, diagnostics[1].Code)
	}
	if len(diagnostics[1].Related) != 1 {
		t.Fatalf("expected duplicate related location, got %+v", diagnostics[1].Related)
	}
}

func TestParseSourceRecoveringSynchronizesAtMissingBlockEnd(t *testing.T) {
	content, diagnostics := ParseSourceRecovering("/workspace/domain.skel", []byte(`domain demo
data User {
    id: string
data Order {
    id: string
}
`))
	if content == nil || len(content.Entries) != 2 {
		t.Fatalf("expected recovery at next top-level declaration: content=%+v diagnostics=%v", content, diagnostics)
	}
	if len(diagnostics) == 0 || diagnostics[0].Suggestion == nil || diagnostics[0].Suggestion.Replacement != "}\n" {
		t.Fatalf("expected missing block-end suggestion, got %v", diagnostics)
	}
}

func TestParseSourceRecoveringSynchronizesAfterBrokenDecorator(t *testing.T) {
	content, diagnostics := ParseSourceRecovering("/workspace/domain.skel", []byte(`domain demo
@desc(
data User {}
`))
	if content == nil || len(content.Entries) != 1 || content.Entries[0].Data == nil {
		t.Fatalf("expected declaration after decorator to survive: content=%+v diagnostics=%v", content, diagnostics)
	}
	if len(diagnostics) != 1 || diagnostics[0].Code != DiagnosticCodeSyntaxEOF {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
}
