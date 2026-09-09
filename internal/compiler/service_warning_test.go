package compiler

import (
	"context"
	"strings"
	"testing"

	"go.yorun.ai/skelc/diagnostic"
)

func TestServiceWarningsAndApiModifiers(t *testing.T) {
	source := Source{Path: "/workspace/api.skel", Content: []byte(`domain demo.order
service LegacyService { method ping {} }
pub service DualService { method ping { noauth } }
api service ClientApiService { method ping {} }
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

func TestApiServiceNameSuffix(t *testing.T) {
	for _, test := range []struct {
		declaration string
		valid       bool
	}{
		{declaration: "api service OrderApiService", valid: true},
		{declaration: "api service OrderService"},
		{declaration: "api service OrderAPIService"},
		{declaration: "api service ApiService"},
		{declaration: "pub service OrderService", valid: true},
		{declaration: "service OrderService", valid: true},
	} {
		t.Run(test.declaration, func(t *testing.T) {
			source := Source{Path: "/workspace/api.skel", Content: []byte("domain demo.order\n" + test.declaration + " { method ping {} }\n")}
			analyzer := NewWorkspaceAnalyzer()
			for range 2 {
				diagnostics, _, err := analyzer.analyze(context.Background(), []Source{source}, true)
				if err != nil {
					t.Fatal(err)
				}
				hasError := false
				for _, d := range diagnostics {
					if d.Severity == DiagnosticSeverityError {
						hasError = true
						if d.Range.Start.Line != 2 || !strings.Contains(d.Message, "ApiService") {
							t.Fatalf("unexpected naming diagnostic: %+v", d)
						}
					}
				}
				if hasError == test.valid {
					t.Fatalf("unexpected diagnostics: %+v", diagnostics)
				}
			}
		})
	}
}
