package parser

import (
	"go.yorun.ai/skelc/internal/loader"
	"path/filepath"
	"testing"
)

func TestParsePanicsWhenDomainNameMissing(t *testing.T) {
	domainSource := &loader.SourceFile{
		FilePath: "/tmp/domain.skel",
		Content:  []byte("domain\n"),
	}

	parser := newParser()
	expectPanicContains(t, "/tmp/domain.skel", func() {
		parser.parseFile(domainSource)
	})
}

func TestParsePanicsForLegacyMethodInOutSyntax(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "legacy.skel")
	source := &loader.SourceFile{
		FilePath: filePath,
		Content: []byte(`domain demo
service UnitService {
    searchUnit {
        in < {
            limit: int
            offset: int
        }
        out > PageResp<Unit>?
    }
}
`),
	}
	parser := newParser()

	expectPanicContains(t, "parse "+filePath+" failed", func() {
		parser.parseFile(source)
	})
}

func TestParsePanicsWhenSkelDomainMismatches(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.skel"), "@desc(\"User domain\")\ndomain demo.user\n")
	writeFile(t, filepath.Join(dir, "user.skel"), "domain demo.account\ndata User { id: string }\n")

	sourceFiles := loader.Load(dir).Files

	parser := newParser()
	expectPanicContains(t, "domain mismatch", func() {
		parser.parseDomainFiles(findDomainFileForTest(t, sourceFiles), sourceFiles)
	})
}

func TestParseDirectorySkelFileWithoutDomainPanics(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.skel"), "@desc(\"User domain\")\ndomain demo.user\n")
	writeFile(t, filepath.Join(dir, "user.skel"), "data User { id: string }\n")

	sourceFiles := loader.Load(dir).Files

	parser := newParser()
	expectPanicContains(t, "missing domain declaration", func() {
		parser.parseDomainFiles(findDomainFileForTest(t, sourceFiles), sourceFiles)
	})
}

func TestParsePanicsWhenDirectorySkelFileDeclaresDomainDecorator(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.skel"), "@desc(\"User domain\")\ndomain demo.user\n")
	writeFile(t, filepath.Join(dir, "user.skel"), "@desc(\"Not allowed\")\ndomain demo.user\ndata User { id: string }\n")

	sourceFiles := loader.Load(dir).Files

	parser := newParser()
	expectPanicContains(t, "domain decorator is only allowed in domain.skel", func() {
		parser.parseDomainFiles(findDomainFileForTest(t, sourceFiles), sourceFiles)
	})
}

func TestParsePanicsWhenDomainFileDeclaresEntries(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.skel"), "@desc(\"User domain\")\ndomain demo.user\ndata User { id: string }\n")
	writeFile(t, filepath.Join(dir, "service.skel"), "domain demo.user\nservice UserService { method ping {} }\n")

	sourceFiles := loader.Load(dir).Files

	parser := newParser()
	expectPanicContains(t, "can only contain domain declaration and @desc", func() {
		parser.parseDomainFiles(findDomainFileForTest(t, sourceFiles), sourceFiles)
	})
}

func TestValidateSource(t *testing.T) {
	ValidateSource("/tmp/demo.skel", []byte("domain demo.user\n"))
}

func TestValidateSourcePanicsForInvalidSyntax(t *testing.T) {
	expectPanicContains(t, "parse /tmp/demo.skel failed", func() {
		ValidateSource("/tmp/demo.skel", []byte("domain {\n"))
	})
}
