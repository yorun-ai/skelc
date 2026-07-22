package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckReusesSingleSyntaxSnapshotPerSource(t *testing.T) {
	directory := t.TempDir()
	files := map[string]string{
		"domain.skel": "domain test.check\n",
		"data.skel":   "domain test.check\npub data User { id: string }\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	workspaceAnalyzer := NewWorkspaceAnalyzer()
	result, err := checkWithAnalyzer(Option{SkelIn: directory}, workspaceAnalyzer)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", result.Diagnostics)
	}
	stats := workspaceAnalyzer.Stats()
	if stats.ParsedSources != 0 || stats.ReusedSources != len(files) {
		t.Fatalf("check did not reuse prepared syntax snapshots: %+v", stats)
	}
}
