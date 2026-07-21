package parser

import (
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"path/filepath"
	"testing"
)

func TestBuildMergedContentUsesDomainFileDomain(t *testing.T) {
	domainFileContent := &grammar.SkelContent{
		Domain: domainContentForTest("demo.user", "User domain"),
	}
	otherContent := &grammar.SkelContent{
		Domain: domainContentForTest("demo.user", ""),
		Entries: []*grammar.SkelEntry{
			{Data: &grammar.Data{Name: identForTest("User")}},
		},
	}

	merged := buildMergedContent(domainFileContent, []*grammar.SkelContent{domainFileContent, otherContent})
	if merged.Domain != domainFileContent.Domain {
		t.Fatal("expected merged domain to come from domain.skel content")
	}
	if len(merged.Entries) != 1 {
		t.Fatalf("unexpected merged entry count: %d", len(merged.Entries))
	}
}

func TestParseSingleSkelAllowsDomainDecorator(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "user.skel")
	singleSource := &loader.SourceFile{
		FilePath: filePath,
		Content: []byte(`@desc("User domain")
domain demo.user
data User { id: string }
`),
	}

	parser := newParser()
	domain := parser.parseFile(singleSource).Model()
	if domain.Name() != "demo.user" {
		t.Fatalf("unexpected domain name: %s", domain.Name())
	}
	if len(domain.Data()) != 1 || domain.Data()[0].Name != "User" {
		t.Fatalf("unexpected data: %+v", domain.Data())
	}
}
