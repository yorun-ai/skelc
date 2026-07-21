package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDirectory(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "domain.skel"), "domain demo.user\n")
	writeFile(t, filepath.Join(dir, "a.skel"), "domain demo.user\ndata User { id: string }\n")
	writeFile(t, filepath.Join(dir, ".hidden.skel"), "domain hidden\n")
	writeFile(t, filepath.Join(dir, "README.md"), "ignored\n")
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	result := Load(dir)

	if !result.IsDir {
		t.Fatal("expected directory result")
	}
	if len(result.Files) != 2 {
		t.Fatalf("unexpected source files: %+v", result.Files)
	}
	if filepath.Base(result.Files[0].FilePath) != "a.skel" || filepath.Base(result.Files[1].FilePath) != "domain.skel" {
		t.Fatalf("source files are not sorted: %+v", result.Files)
	}
	if len(result.Warnings) != 3 {
		t.Fatalf("unexpected warnings: %+v", result.Warnings)
	}
}

func TestLoadSingleFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "user.skel")
	writeFile(t, filePath, "domain demo.user\ndata User { id: string }\n")

	result := Load(filePath)

	if result.IsDir {
		t.Fatal("did not expect directory result")
	}
	if len(result.Files) != 1 || result.Files[0].FilePath != filePath {
		t.Fatalf("unexpected source files: %+v", result.Files)
	}
	if string(result.Files[0].Content) != "domain demo.user\ndata User { id: string }\n" {
		t.Fatalf("unexpected source content: %q", result.Files[0].Content)
	}
}

func TestLoadPanicsWhenDomainFileMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "service.skel"), "domain demo.user\n")

	expectPanicContains(t, "domain.skel not found", func() { Load(dir) })
}

func TestLoadRejectsNonSkelFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "README.md")
	writeFile(t, filePath, "ignored\n")

	expectPanicContains(t, "is not a .skel file", func() { Load(filePath) })
}

func TestClassifyFile(t *testing.T) {
	tests := map[string]string{
		".hidden.skel": "hidden",
		"domain.skel":  "domain",
		"data.skel":    "skel",
		"README.md":    "other",
	}
	for name, expected := range tests {
		if actual := classifyFile(name); actual != expected {
			t.Fatalf("classifyFile(%q) = %q, want %q", name, actual, expected)
		}
	}
}

func expectPanicContains(t *testing.T, expected string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic containing %q", expected)
		}
		if !strings.Contains(fmt.Sprint(recovered), expected) {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()
	fn()
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
