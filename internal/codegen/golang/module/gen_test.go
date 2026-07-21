package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateIncludesVersionedGoImports(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "golang")
	Generate(Option{
		Out:         outputDir,
		Module:      "go.yorun.ai/app/vine/demo/skeled/booker",
		VineVersion: "v1.2.3",
		Imports: map[string]string{
			"user": "go.yorun.ai/app/vine/demo/skeled/userpub@v0.0.0-00010101000000-000000000000",
		},
	})

	content := readGoModForTest(t, outputDir)
	if !strings.Contains(content, "go.yorun.ai/app/vine/demo/skeled/userpub v0.0.0-00010101000000-000000000000") {
		t.Fatalf("expected versioned go import require:\n%s", content)
	}
}

func TestGenerateIncludesDefaultVersionForGoImports(t *testing.T) {
	outputDir := filepath.Join(t.TempDir(), "golang")
	Generate(Option{
		Out:         outputDir,
		Module:      "go.yorun.ai/app/vine/demo/skeled/booker",
		VineVersion: "v1.2.3",
		Imports: map[string]string{
			"user": "go.yorun.ai/app/vine/demo/skeled/userpub",
		},
	})

	content := readGoModForTest(t, outputDir)
	if !strings.Contains(content, "go.yorun.ai/app/vine/demo/skeled/userpub v0.0.0-00010101000000-000000000000") {
		t.Fatalf("expected default go import require:\n%s", content)
	}
}

func readGoModForTest(t *testing.T, outputDir string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(outputDir, goModFilename))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	return string(content)
}
