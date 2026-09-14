package skelc_test

import (
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/model"
)

func TestImportValidationOrderIsDeterministic(t *testing.T) {
	domain := model.NewDomainFromSpec(model.DomainSpec{Name: "demo.test"})
	imports := map[string]string{"z": "example.com/z@", "a": "example.com/a@"}
	out := filepath.Join(t.TempDir(), "generated")
	for range 30 {
		goErr := skelc.GenerateGolang(domain, skelc.GolangOption{Out: out, CompilerVersion: "v0.0.0-dev", Imports: imports})
		tsErr := skelc.GenerateTypeScript(domain, skelc.TypeScriptOption{Out: out, ApiOnly: true, Imports: imports})
		for _, err := range []error{goErr, tsErr} {
			if err == nil || !strings.Contains(err.Error(), "example.com/a@") {
				t.Fatalf("unstable first error: %v", err)
			}
		}
	}
}
