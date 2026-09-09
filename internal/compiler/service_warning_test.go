package compiler

import (
	"context"
	"testing"

	"go.yorun.ai/skelc/diagnostic"
)

func TestServiceWarningsAndApiModifiers(t *testing.T) {
	source := Source{Path: "/workspace/api.skel", Content: []byte(`domain demo.order
service LegacyService { method ping {} }
pub service DualService { method ping { noauth } }
api service ClientService { method ping {} }
pub service BackendService { method ping {} }
`)}
	analyzer := NewWorkspaceAnalyzer()
	for range 2 {
		diagnostics, _, err := analyzer.analyze(context.Background(), []Source{source}, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(diagnostics) != 2 || diagnostics[0].Code != diagnostic.CodeServiceModifier || diagnostics[1].Code != diagnostic.CodeServiceClientRules {
			t.Fatalf("unexpected diagnostics: %v", diagnostics)
		}
		for _, d := range diagnostics {
			if d.Severity != DiagnosticSeverityWarning || d.Range.Start.Line < 2 {
				t.Fatalf("invalid warning: %+v", d)
			}
		}
	}
}
