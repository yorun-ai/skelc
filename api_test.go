package skelc_test

import (
	"encoding/json"
	"fmt"
	"go/ast"
	stdparser "go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/diagnostic"
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

func TestDiagnosticAliasesMatchPublicDiagnosticPackage(t *testing.T) {
	if skelc.DiagnosticCodeSyntaxUnexpected != diagnostic.CodeSyntaxUnexpected {
		t.Fatalf("syntax diagnostic code alias = %q, want %q", skelc.DiagnosticCodeSyntaxUnexpected, diagnostic.CodeSyntaxUnexpected)
	}
	if skelc.DiagnosticCodeSemanticDuplicate != diagnostic.CodeSemanticDuplicate {
		t.Fatalf("semantic diagnostic code alias = %q, want %q", skelc.DiagnosticCodeSemanticDuplicate, diagnostic.CodeSemanticDuplicate)
	}
	if skelc.DiagnosticCodeLoaderUnsupported != diagnostic.CodeLoaderUnsupported {
		t.Fatalf("loader diagnostic code alias = %q, want %q", skelc.DiagnosticCodeLoaderUnsupported, diagnostic.CodeLoaderUnsupported)
	}
	if skelc.DiagnosticSeverityWarning != diagnostic.SeverityWarning {
		t.Fatalf("warning severity alias = %q, want %q", skelc.DiagnosticSeverityWarning, diagnostic.SeverityWarning)
	}
}

func TestSchemaJSONFacadeTypes(t *testing.T) {
	var entries []*skelc.SchemaEntry
	if err := json.Unmarshal([]byte(`[{"pub":true,"name":"User","type":"data","skelName":"demo.user.User"}]`), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].SkelName != "demo.user.User" {
		t.Fatalf("unexpected schema entries: %+v", entries)
	}

	var declaration skelc.SchemaDeclaration
	if err := json.Unmarshal([]byte(`{"pub":true,"name":"Status","type":"enum","skelName":"demo.user.Status","enum":{"items":[{"name":"ACTIVE"}]}}`), &declaration); err != nil {
		t.Fatal(err)
	}
	if declaration.Enum == nil || len(declaration.Enum.Items) != 1 {
		t.Fatalf("unexpected schema declaration: %+v", declaration)
	}

	var snapshot skelc.SchemaSnapshot
	if err := json.Unmarshal([]byte(`{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo.user","declarations":[]}`), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Format != skelc.SchemaFormat || snapshot.FormatVersion != skelc.SchemaFormatVersion {
		t.Fatalf("unexpected schema snapshot: %+v", snapshot)
	}

	var report skelc.SchemaDiffReport
	if err := json.Unmarshal([]byte(`{"compatible":true,"baselineDomain":"demo.user","candidateDomain":"demo.user","summary":{"breaking":0,"dangerous":1,"compatible":0},"changes":[{"code":"method.auth.changed","change":"MODIFIED","impact":"DANGEROUS","symbol":"demo.user.UserService.get"}]}`), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Changes) != 1 || report.Changes[0].Change != skelc.SchemaChangeModified ||
		report.Changes[0].Impact != skelc.SchemaImpactDangerous {
		t.Fatalf("unexpected schema diff report: %+v", report)
	}
}

func TestPublicPackagesRespectDependencyBoundaries(t *testing.T) {
	const internalPrefix = "go.yorun.ai/skelc/internal/"
	rules := []struct {
		directory string
		forbidden func(string) bool
	}{
		{
			directory: "internal/parser",
			forbidden: func(path string) bool {
				return strings.HasPrefix(path, internalPrefix) && path != internalPrefix+"parser/grammar"
			},
		},
		{
			directory: "internal/analyzer",
			forbidden: func(path string) bool {
				for _, prefix := range []string{"compiler", "hasher", "loader", "lsp", "codegen"} {
					if strings.HasPrefix(path, internalPrefix+prefix) {
						return true
					}
				}
				return false
			},
		},
		{
			directory: "internal/hasher",
			forbidden: func(path string) bool { return strings.HasPrefix(path, internalPrefix) },
		},
		{
			directory: "model",
			forbidden: func(path string) bool { return strings.HasPrefix(path, internalPrefix) },
		},
		{
			directory: "diagnostic",
			forbidden: func(path string) bool { return strings.HasPrefix(path, internalPrefix) },
		},
		{
			directory: "schema",
			forbidden: func(path string) bool { return strings.HasPrefix(path, internalPrefix) },
		},
		{
			directory: "internal/codegen/common",
			forbidden: targetCodegenImport,
		},
		{
			directory: "internal/codegen/output",
			forbidden: targetCodegenImport,
		},
	}

	for _, rule := range rules {
		t.Run(filepath.ToSlash(rule.directory), func(t *testing.T) {
			inspectProductionImports(t, rule.directory, rule.forbidden)
		})
	}
}

func targetCodegenImport(path string) bool {
	for _, target := range []string{"golang", "skeleton", "typescript"} {
		if strings.HasPrefix(path, "go.yorun.ai/skelc/internal/codegen/"+target) {
			return true
		}
	}
	return false
}

func inspectProductionImports(t *testing.T, directory string, forbidden func(string) bool) {
	t.Helper()
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := stdparser.ParseFile(token.NewFileSet(), path, nil, stdparser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, declaration := range file.Decls {
			importDecl, ok := declaration.(*ast.GenDecl)
			if !ok || importDecl.Tok != token.IMPORT {
				continue
			}
			for _, spec := range importDecl.Specs {
				pathValue, err := strconv.Unquote(spec.(*ast.ImportSpec).Path.Value)
				if err != nil {
					return err
				}
				if forbidden(pathValue) {
					t.Errorf("%s imports forbidden dependency %s", path, pathValue)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
	assertTestFileStartsWithGeneratedMarker(t, filepath.Join(goOut, "doc.go"))
	assertTestFileMissing(t, filepath.Join(goOut, ".skelc-manifest.json"))
}

func TestCompileTargetsWithImportedDomainNamedTypeReferences(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "domain.skel"), "domain base")
	writeTestFile(t, filepath.Join(baseDir, "types.skel"), `
domain base

pub enum ItemType {
    STANDARD
}

pub data Item {
    type: ItemType
    types: list<ItemType>
    itemsByType: map<ItemType, Item>
}

pub config ItemConfig eternal {
    defaultType: ItemType
}

pub event ItemCreatedEvent {
    payload {
        item: Item
    }
}

pub actor BaseActor {
    via client {}
    auth {
        credential {
            subject: string
        }
        info {
            item: Item
        }
    }
}

pub resource ItemResource {
    check byItem {
        input {
            item: Item
        }
    }
    action read
}

pub service BaseService {
    for BaseActor

    method getItem {
        input {
            type: ItemType
        }
        output Item
    }
}

task SyncItemTask {
    trigger manually {
        input {
            item: Item
        }
    }
}
`)
	appDir := t.TempDir()
	writeTestFile(t, filepath.Join(appDir, "domain.skel"), "domain app")
	writeTestFile(t, filepath.Join(appDir, "types.skel"), `
domain app

import base

data AppItem {
    item: base.Item
}
`)

	input := skelc.Input{SkelIn: appDir, SkelImports: map[string]string{"base": baseDir}}
	tests := []struct {
		name    string
		compile func() error
	}{
		{
			name: "Go",
			compile: func() error {
				_, err := skelc.CompileGolang(input, skelc.GolangOption{
					Out:     filepath.Join(t.TempDir(), "golang"),
					Imports: map[string]string{"base": "example.com/basepub"},
				})
				return err
			},
		},
		{
			name: "TypeScript",
			compile: func() error {
				_, err := skelc.CompileTypeScript(input, skelc.TypeScriptOption{
					Out:     filepath.Join(t.TempDir(), "typescript"),
					Imports: map[string]string{"base": "@example/base"},
				})
				return err
			},
		},
		{
			name: "Skel",
			compile: func() error {
				_, err := skelc.CompileSkeleton(input, skelc.SkeletonOption{
					Out:     filepath.Join(t.TempDir(), "skeleton"),
					PubOnly: true,
				})
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.compile(); err != nil {
				t.Fatalf("compile %s with imported named type references: %v", test.name, err)
			}
		})
	}
}

func TestCompileGolangAnalyzesTransitiveImportsButGeneratesOnlyTarget(t *testing.T) {
	baseDir := t.TempDir()
	writeTestFile(t, filepath.Join(baseDir, "domain.skel"), "domain base")
	writeTestFile(t, filepath.Join(baseDir, "types.skel"), `
domain base

pub enum UserStatus {
    ACTIVE
}
`)
	userDir := t.TempDir()
	writeTestFile(t, filepath.Join(userDir, "domain.skel"), "domain user")
	writeTestFile(t, filepath.Join(userDir, "types.skel"), `
domain user

import base

pub data User {
    status: base.UserStatus
}
`)
	appDir := t.TempDir()
	writeTestFile(t, filepath.Join(appDir, "domain.skel"), "domain app")
	writeTestFile(t, filepath.Join(appDir, "types.skel"), `
domain app

import user

data AppUser {
    user: user.User
}
`)

	goOut := filepath.Join(t.TempDir(), "generated")
	_, err := skelc.CompileGolang(
		skelc.Input{
			SkelIn: appDir,
			SkelImports: map[string]string{
				"base": baseDir,
				"user": userDir,
			},
		},
		skelc.GolangOption{
			Out:     goOut,
			Imports: map[string]string{"user": "example.com/userpub"},
		},
	)
	if err != nil {
		t.Fatalf("compile Go with transitive imports: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(goOut, "data.go"))
	if err != nil {
		t.Fatalf("read generated target data: %v", err)
	}
	generated := string(content)
	if !strings.Contains(generated, "type AppUser struct") {
		t.Fatalf("target data was not generated:\n%s", generated)
	}
	if strings.Contains(generated, "type User struct") || strings.Contains(generated, "type UserStatus ") {
		t.Fatalf("dependency declarations were generated into target output:\n%s", generated)
	}
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
	assertTestFileStartsWithGeneratedMarker(t, filepath.Join(tsOut, "index.ts"))
	assertTestFileMissing(t, filepath.Join(tsOut, ".skelc-manifest.json"))
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
	assertTestFileStartsWithGeneratedMarker(t, filepath.Join(skelOut, "domain.skel"))
	assertTestFileMissing(t, filepath.Join(skelOut, ".skelc-manifest.json"))
	if _, err := skelc.Parse(skelc.Input{SkelIn: skelOut}); err != nil {
		t.Fatalf("parse generated Skel with ownership marker: %v", err)
	}
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

func TestCompileNormalizesGenerationOptionsBeforeReadingInput(t *testing.T) {
	missingInput := skelc.Input{SkelIn: filepath.Join(t.TempDir(), "missing")}
	tests := []struct {
		name     string
		compile  func() error
		expected string
	}{
		{
			name: "Go",
			compile: func() error {
				_, err := skelc.CompileGolang(missingInput, skelc.GolangOption{})
				return err
			},
			expected: "Go output is required",
		},
		{
			name: "TypeScript",
			compile: func() error {
				_, err := skelc.CompileTypeScript(missingInput, skelc.TypeScriptOption{})
				return err
			},
			expected: "TypeScript output is required",
		},
		{
			name: "Skel",
			compile: func() error {
				_, err := skelc.CompileSkeleton(missingInput, skelc.SkeletonOption{})
				return err
			},
			expected: "Skel output is required",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.compile(); err == nil || !strings.Contains(err.Error(), test.expected) {
				t.Fatalf("expected %q before input loading, got %v", test.expected, err)
			}
		})
	}
}

func TestGeneratorsReturnErrorsForMalformedProgrammaticModels(t *testing.T) {
	domain := model.NewDomainFromSpec(model.DomainSpec{
		Name: "demo.invalid",
		Data: []*model.Data{{
			Name: "Broken",
			Pub:  true,
			Members: []*model.DataMember{{
				Name: "value",
				Type: &model.Type{Kind: model.TypeKind(999)},
			}},
		}},
	})
	tests := []struct {
		name     string
		generate func() error
	}{
		{
			name: "Go",
			generate: func() error {
				return skelc.GenerateGolang(domain, skelc.GolangOption{Out: filepath.Join(t.TempDir(), "generated")})
			},
		},
		{
			name: "TypeScript",
			generate: func() error {
				return skelc.GenerateTypeScript(domain, skelc.TypeScriptOption{Out: t.TempDir()})
			},
		},
		{
			name: "Skel",
			generate: func() error {
				return skelc.GenerateSkeleton(domain, skelc.SkeletonOption{Out: t.TempDir(), PubOnly: true})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.generate()
			if err == nil || !strings.Contains(err.Error(), "unsupported type kind 999") {
				t.Fatalf("expected malformed model error, got %v", err)
			}
		})
	}
}

func TestGeneratorsReturnErrorsForMalformedNestedModels(t *testing.T) {
	domains := []struct {
		name     string
		domain   *model.Domain
		expected string
	}{
		{
			name: "incomplete actor auth",
			domain: model.NewDomainFromSpec(model.DomainSpec{
				Name: "demo.invalid", Actors: []*model.Actor{{Name: "Client", AuthEnabled: true}},
			}),
			expected: "incomplete auth support",
		},
		{
			name: "nil import",
			domain: model.NewDomainFromSpec(model.DomainSpec{
				Name: "demo.invalid", Imports: []*model.Import{nil},
			}),
			expected: "nil import",
		},
	}
	for _, malformed := range domains {
		t.Run(malformed.name, func(t *testing.T) {
			generators := []struct {
				name     string
				generate func() error
			}{
				{name: "Go", generate: func() error {
					return skelc.GenerateGolang(malformed.domain, skelc.GolangOption{Out: filepath.Join(t.TempDir(), "generated")})
				}},
				{name: "TypeScript", generate: func() error {
					return skelc.GenerateTypeScript(malformed.domain, skelc.TypeScriptOption{Out: t.TempDir()})
				}},
				{name: "Skel", generate: func() error {
					return skelc.GenerateSkeleton(malformed.domain, skelc.SkeletonOption{Out: t.TempDir(), PubOnly: true})
				}},
			}
			for _, generator := range generators {
				t.Run(generator.name, func(t *testing.T) {
					err := generator.generate()
					if err == nil || !strings.Contains(err.Error(), malformed.expected) {
						t.Fatalf("expected error containing %q, got %v", malformed.expected, err)
					}
				})
			}
		})
	}
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

func assertTestFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file %s to be missing: %v", path, err)
	}
}

func assertTestFileStartsWithGeneratedMarker(t *testing.T, path string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}
	const marker = "// Code generated by skelc. DO NOT EDIT.\n"
	if !strings.HasPrefix(string(content), marker) {
		t.Fatalf("expected %s to start with %q, got %q", path, marker, content)
	}
}
