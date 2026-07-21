package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSkelcFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "domain.skel")
	writeCLIFile(t, path, "  domain demo.user  \n")

	result := Run([]string{"format", "--skel-in", dir})
	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected result: %+v", result)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "domain demo.user\n" {
		t.Fatalf("unexpected formatted content: %q", content)
	}
}

func TestRunSkelcFormatChecksAllFilesBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	domainPath := filepath.Join(dir, "domain.skel")
	writeCLIFile(t, domainPath, "  domain demo.user\n")
	writeCLIFile(t, filepath.Join(dir, "invalid.skel"), "domain demo.user\ndata Invalid { value: }\n")

	result := Run([]string{"format", "--skel-in", dir})
	if result.ExitCode == ExitCodeSuccess {
		t.Fatal("expected format failure")
	}
	content, err := os.ReadFile(domainPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "  domain demo.user\n" {
		t.Fatalf("valid file changed before validation completed: %q", content)
	}
}
