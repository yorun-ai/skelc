package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

func parseDomain(t *testing.T, files map[string]string) *model.Domain {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		if name != loader.DomainFileName && !strings.HasPrefix(strings.TrimLeft(content, "\n\t "), "domain ") {
			content = "domain demo.user\n" + content
		}
		writeFile(t, filepath.Join(dir, name), content)
	}

	loadResult, err := loader.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	sourceFiles := loadResult.Files

	parser := newParser()
	domain, err := parser.parseDomainFiles(findDomainFileForTest(t, sourceFiles), sourceFiles)
	if err != nil {
		t.Fatalf("parse domain: %v", err)
	}
	return domain.Model()
}

func findDataByName(t *testing.T, domain *model.Domain, name string) *model.Data {
	t.Helper()

	for _, data := range domain.Data() {
		if data.Name == name {
			return data
		}
	}
	t.Fatalf("data %s not found", name)
	return nil
}

func findServiceByName(t *testing.T, domain *model.Domain, name string) *model.Service {
	t.Helper()

	for _, service := range domain.Services() {
		if service.Name == name {
			return service
		}
	}
	t.Fatalf("service %s not found", name)
	return nil
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func findDomainFileForTest(t *testing.T, sourceFiles []*loader.SourceFile) *loader.SourceFile {
	t.Helper()

	for _, sourceFile := range sourceFiles {
		if filepath.Base(sourceFile.FilePath) == loader.DomainFileName {
			return sourceFile
		}
	}
	t.Fatal("domain.skel not found")
	return nil
}

func identForTest(value string) *grammar.Identifier {
	return &grammar.Identifier{Value: value}
}

func domainContentForTest(name string, description string) *grammar.DomainContent {
	parts := strings.Split(name, ".")
	identParts := make([]*grammar.Identifier, 0, len(parts))
	for _, part := range parts {
		identParts = append(identParts, identForTest(part))
	}

	return &grammar.DomainContent{
		Description: description,
		Name: &grammar.QualifiedName{
			Parts: identParts,
		},
	}
}
func expectErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}
