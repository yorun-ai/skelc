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

func TestRunSkelcFormatPreservesFileMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "domain.skel")
	writeCLIFile(t, path, "  domain demo.user  \n")
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}

	result := Run([]string{"format", "--skel-in", dir})
	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected result: %+v", result)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected formatted file mode: %o", info.Mode().Perm())
	}
}

func TestRunSkelcFormatPreservesInputSymlink(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.skel")
	linkPath := filepath.Join(dir, "domain.skel")
	writeCLIFile(t, targetPath, "  domain demo.user  \n")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	result := Run([]string{"format", "--skel-in", linkPath})
	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected result: %+v", result)
	}
	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("format replaced the input symlink")
	}
	assertFileExact(t, targetPath, "domain demo.user\n")
}

func assertFileExact(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expected {
		t.Fatalf("unexpected content for %s: %q", path, content)
	}
}
