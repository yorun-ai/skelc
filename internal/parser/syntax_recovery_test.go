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

func TestParseSourceRecoveringKeepsDecoratorAfterMissingBlockEnd(t *testing.T) {
	content, diagnostics := ParseSourceRecovering("/workspace/domain.skel", []byte(`domain demo
data User {
    id: string
@desc("order")
data Order {}
`))
	if content == nil || len(content.Entries) != 2 {
		t.Fatalf("expected both declarations, got content=%+v diagnostics=%v", content, diagnostics)
	}
	order := content.Entries[1].Data
	if order == nil || len(order.Decorators) != 1 || order.Decorators[0].Name.Value != "desc" {
		t.Fatalf("expected decorator to remain attached to Order: %+v", order)
	}
	if len(diagnostics) != 1 || diagnostics[0].Suggestion == nil || diagnostics[0].Suggestion.Replacement != "}\n" {
		t.Fatalf("expected one missing block-end diagnostic, got %v", diagnostics)
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

func TestParseSourceRecoveringKeepsTopLevelKeywordsInsideDeclarations(t *testing.T) {
	source := []byte(`domain demo
data Example {
    data: string
    @desc("""first line
data NotADeclaration {}
last line""")
    description: string
}
data Actual {}
`)
	content, diagnostics := ParseSourceRecovering("/workspace/domain.skel", source)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if content == nil || len(content.Entries) != 2 {
		t.Fatalf("expected two declarations, got %+v", content)
	}
	if got := content.Entries[0].Data.Members; len(got) != 2 || got[0].Name.Value != "data" {
		t.Fatalf("unexpected data members: %+v", got)
	}
}

func TestParseSourceRecoveringPreservesFragmentPositionsAndDecorators(t *testing.T) {
	source := []byte(`domain demo

@desc("order")
data Order {
    id string
}
`)
	content, diagnostics := ParseSourceRecovering("/workspace/domain.skel", source)
	if content == nil || len(content.Entries) != 1 {
		t.Fatalf("expected recovered declaration, got %+v", content)
	}
	entry := content.Entries[0]
	if entry.Pos.Line != 3 || entry.Data.Pos.Line != 4 || len(entry.Data.Decorators) != 1 {
		t.Fatalf("fragment positions or decorators changed: %+v", entry)
	}
	if len(diagnostics) != 1 || diagnostics[0].Position.Line != 5 {
		t.Fatalf("expected syntax error on original line 5, got %v", diagnostics)
	}
}

func TestParseSourceRecoveringPreservesDeclarationOrderingRules(t *testing.T) {
	content, diagnostics := ParseSourceRecovering("/workspace/domain.skel", []byte(`domain demo
data User {}
import shared.types
domain other
data Order {}
`))
	if content == nil || len(content.Entries) != 2 || len(content.Imports) != 0 {
		t.Fatalf("unexpected recovered content: %+v", content)
	}
	if len(diagnostics) != 2 {
		t.Fatalf("expected import and domain ordering diagnostics, got %v", diagnostics)
	}
	if diagnostics[0].Position.Line != 3 || diagnostics[1].Position.Line != 4 {
		t.Fatalf("unexpected diagnostic positions: %v", diagnostics)
	}
}
