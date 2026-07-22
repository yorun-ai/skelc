package skelc_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/model"
)

func ExampleParse() {
	skelDir, err := os.MkdirTemp("", "skelc-example-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(skelDir)

	if err := os.WriteFile(filepath.Join(skelDir, "domain.skel"), []byte("domain demo.user"), 0o644); err != nil {
		panic(err)
	}

	result, err := skelc.Parse(skelc.Input{SkelIn: skelDir})
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Domain.Name())

	// Output: demo.user
}

func TestParseExposesSemanticModel(t *testing.T) {
	skelDir := t.TempDir()
	writeTestFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.user")

	result, err := skelc.Parse(skelc.Input{SkelIn: skelDir})
	if err != nil {
		t.Fatalf("parse Skel: %v", err)
	}
	var domain *model.Domain = result.Domain
	if domain.Name() != "demo.user" {
		t.Fatalf("unexpected domain: %s", domain.Name())
	}
}

func TestPublicOptionsRejectEmptyImportMappings(t *testing.T) {
	if _, err := skelc.Parse(skelc.Input{SkelIn: "input.skel", SkelImports: map[string]string{"demo.user": ""}}); err == nil {
		t.Fatal("expected empty Skel import path error")
	}
	domain := model.NewDomainFromSpec(model.DomainSpec{Name: "demo.test"})
	if err := skelc.GenerateGolang(domain, skelc.GolangOption{
		Out: filepath.Join(t.TempDir(), "golang"), Imports: map[string]string{"demo.user": ""},
	}); err == nil {
		t.Fatal("expected empty Go import path error")
	}
	if err := skelc.GenerateTypeScript(domain, skelc.TypeScriptOption{
		Out: filepath.Join(t.TempDir(), "typescript"), Imports: map[string]string{"demo.user": ""},
	}); err == nil {
		t.Fatal("expected empty TypeScript import path error")
	}
}

func TestParseReturnsStructuredWarnings(t *testing.T) {
	skelDir := t.TempDir()
	writeTestFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.user")
	writeTestFile(t, filepath.Join(skelDir, ".ignored.skel"), "domain ignored")

	result, err := skelc.Parse(skelc.Input{SkelIn: skelDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != "loader.ignored-hidden-file" || diagnostic.Severity != skelc.DiagnosticSeverityWarning {
		t.Fatalf("unexpected warning: %+v", diagnostic)
	}
}

func TestGenerateGolang(t *testing.T) {
	skelDir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "generated")
	writeTestFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.user")

	result, err := skelc.CompileGolang(
		skelc.Input{SkelIn: skelDir},
		skelc.GolangOption{Out: goOut, CompilerVersion: "v1.2.3"},
	)
	if err != nil {
		t.Fatalf("generate Go: %v", err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	assertTestFileExists(t, filepath.Join(goOut, "doc.go"))
	assertTestFileExists(t, filepath.Join(goOut, "schema.go"))
}

func TestGenerateTargetsShareParsedDomain(t *testing.T) {
	skelDir := t.TempDir()
	writeTestFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.user")

	parsed, err := skelc.Parse(skelc.Input{SkelIn: skelDir})
	if err != nil {
		t.Fatalf("parse Skel: %v", err)
	}
	goOut := filepath.Join(t.TempDir(), "golang")
	if err := skelc.GenerateGolang(parsed.Domain, skelc.GolangOption{Out: goOut}); err != nil {
		t.Fatalf("generate Go: %v", err)
	}
	tsOut := filepath.Join(t.TempDir(), "typescript")
	if err := skelc.GenerateTypeScript(parsed.Domain, skelc.TypeScriptOption{Out: tsOut}); err != nil {
		t.Fatalf("generate TypeScript: %v", err)
	}
	assertTestFileExists(t, filepath.Join(goOut, "schema.go"))
	assertTestFileExists(t, filepath.Join(tsOut, "index.ts"))
}

func TestGenerateTypeScript(t *testing.T) {
	skelDir := t.TempDir()
	tsOut := filepath.Join(t.TempDir(), "generated")
	writeTestFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.user")

	_, err := skelc.CompileTypeScript(
		skelc.Input{SkelIn: skelDir},
		skelc.TypeScriptOption{Out: tsOut},
	)
	if err != nil {
		t.Fatalf("generate TypeScript: %v", err)
	}
	assertTestFileExists(t, filepath.Join(tsOut, "index.ts"))
}

func TestGenerateSkeleton(t *testing.T) {
	skelDir := t.TempDir()
	skelOut := filepath.Join(t.TempDir(), "generated")
	writeTestFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.user")

	_, err := skelc.CompileSkeleton(
		skelc.Input{SkelIn: skelDir},
		skelc.SkeletonOption{Out: skelOut, PubOnly: true},
	)
	if err != nil {
		t.Fatalf("generate Skel: %v", err)
	}
	assertTestFileExists(t, filepath.Join(skelOut, "domain.skel"))
}

func TestGenerateGolangReturnsErrorBeforeCleaningOutput(t *testing.T) {
	skelDir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "generated")
	writeTestFile(t, filepath.Join(skelDir, "invalid.skel"), "data User { id: string }")
	oldFile := filepath.Join(goOut, "old.go")
	writeTestFile(t, oldFile, "old")

	_, err := skelc.CompileGolang(
		skelc.Input{SkelIn: skelDir},
		skelc.GolangOption{Out: goOut},
	)
	if err == nil {
		t.Fatal("expected generation error")
	}
	assertTestFileExists(t, oldFile)
}

func TestGeneratorsReturnErrorsForMissingExternalImportMappings(t *testing.T) {
	userDir := t.TempDir()
	writeTestFile(t, filepath.Join(userDir, "domain.skel"), "domain demo.user")
	writeTestFile(t, filepath.Join(userDir, "user.skel"), "domain demo.user\npub data User { id: string }")
	orderDir := t.TempDir()
	writeTestFile(t, filepath.Join(orderDir, "domain.skel"), "domain demo.order")
	writeTestFile(t, filepath.Join(orderDir, "order.skel"), "domain demo.order\nimport demo.user\ndata Order { user: user.User }")

	parsed, err := skelc.Parse(skelc.Input{
		SkelIn:      orderDir,
		SkelImports: map[string]string{"demo.user": userDir},
	})
	if err != nil {
		t.Fatalf("parse imported domain: %v", err)
	}

	goErr := skelc.GenerateGolang(parsed.Domain, skelc.GolangOption{Out: filepath.Join(t.TempDir(), "golang")})
	if goErr == nil || !strings.Contains(goErr.Error(), "missing Go import for domain demo.user") {
		t.Fatalf("expected missing Go import error, got %v", goErr)
	}
	tsErr := skelc.GenerateTypeScript(parsed.Domain, skelc.TypeScriptOption{Out: filepath.Join(t.TempDir(), "typescript")})
	if tsErr == nil || !strings.Contains(tsErr.Error(), "missing TypeScript import for domain demo.user") {
		t.Fatalf("expected missing TypeScript import error, got %v", tsErr)
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}

func assertTestFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s: %v", path, err)
	}
}
