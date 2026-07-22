package parser

import (
	"path/filepath"
	"testing"

	"go.yorun.ai/skelc/internal/loader"
)

func TestParseReturnsErrorWhenDomainNameMissing(t *testing.T) {
	source := &loader.SourceFile{FilePath: "/tmp/domain.skel", Content: []byte("domain\n")}
	_, err := parseFileWithImports(source, nil)
	expectErrorContains(t, err, "/tmp/domain.skel")
}

func TestParseReturnsErrorForLegacyMethodInOutSyntax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.skel")
	source := &loader.SourceFile{FilePath: path, Content: []byte(`domain demo
service UnitService {
    searchUnit {
        in < {
            limit: int
            offset: int
        }
        out > PageResp<Unit>?
    }
}
`)}
	_, err := parseFileWithImports(source, nil)
	expectErrorContains(t, err, "parse "+path+" failed")
}

func TestParseReturnsErrorWhenSkelDomainMismatches(t *testing.T) {
	files := validationFilesForTest(t,
		"@desc(\"User domain\")\ndomain demo.user\n",
		"domain demo.account\ndata User { id: string }\n",
	)
	_, err := parseDomainFilesWithImports(findDomainFileForTest(t, files), files, nil)
	expectErrorContains(t, err, "domain mismatch")
}

func TestParseDirectorySkelFileWithoutDomainReturnsError(t *testing.T) {
	files := validationFilesForTest(t,
		"@desc(\"User domain\")\ndomain demo.user\n",
		"data User { id: string }\n",
	)
	_, err := parseDomainFilesWithImports(findDomainFileForTest(t, files), files, nil)
	expectErrorContains(t, err, "missing domain declaration")
}

func TestParseReturnsErrorWhenDirectorySkelFileDeclaresDomainDecorator(t *testing.T) {
	files := validationFilesForTest(t,
		"@desc(\"User domain\")\ndomain demo.user\n",
		"@desc(\"Not allowed\")\ndomain demo.user\ndata User { id: string }\n",
	)
	_, err := parseDomainFilesWithImports(findDomainFileForTest(t, files), files, nil)
	expectErrorContains(t, err, "domain decorator is only allowed in domain.skel")
}

func TestParseReturnsErrorWhenDomainFileDeclaresEntries(t *testing.T) {
	files := validationFilesForTest(t,
		"@desc(\"User domain\")\ndomain demo.user\ndata User { id: string }\n",
		"domain demo.user\nservice UserService { method ping {} }\n",
	)
	_, err := parseDomainFilesWithImports(findDomainFileForTest(t, files), files, nil)
	expectErrorContains(t, err, "can only contain domain declaration and @desc")
}

func TestValidateSource(t *testing.T) {
	if err := ValidateSource("/tmp/demo.skel", []byte("domain demo.user\n")); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSourceReturnsErrorForInvalidSyntax(t *testing.T) {
	err := ValidateSource("/tmp/demo.skel", []byte("domain {\n"))
	expectErrorContains(t, err, "parse /tmp/demo.skel failed")
}

func validationFilesForTest(t *testing.T, domain, source string) []*loader.SourceFile {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.skel"), domain)
	writeFile(t, filepath.Join(dir, "user.skel"), source)
	result, err := loader.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return result.Files
}
