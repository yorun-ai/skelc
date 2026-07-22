package common

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagedOutputPreservesUnmanagedFilesAndRemovesUnchangedStaleFiles(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	writeOutputTestFile(t, filepath.Join(target, "user.go"), "user")

	first := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(first.StageDir(), "old.go"), "old")
	writeOutputTestFile(t, filepath.Join(first.StageDir(), "nested", "keep.go"), "first")
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}

	second := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(second.StageDir(), "nested", "keep.go"), "second")
	if err := second.Commit(); err != nil {
		t.Fatal(err)
	}

	assertOutputTestContent(t, filepath.Join(target, "user.go"), "user")
	assertOutputTestContent(t, filepath.Join(target, "nested", "keep.go"), "second")
	assertOutputTestMissing(t, filepath.Join(target, "old.go"))
	assertOutputTestContent(t, filepath.Join(target, outputManifestName), `"nested/keep.go"`)
}

func TestManagedOutputPreservesModifiedStaleGeneratedFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	first := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(first.StageDir(), "old.go"), "generated")
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	writeOutputTestFile(t, filepath.Join(target, "old.go"), "user modified")

	second := newOutputTestTransaction(t, target)
	if err := second.Commit(); err != nil {
		t.Fatal(err)
	}

	assertOutputTestContent(t, filepath.Join(target, "old.go"), "user modified")
}

func TestManagedOutputAbortLeavesTargetUnchanged(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	writeOutputTestFile(t, filepath.Join(target, "existing.go"), "existing")
	output := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(output.StageDir(), "new.go"), "new")
	output.Abort()

	assertOutputTestContent(t, filepath.Join(target, "existing.go"), "existing")
	assertOutputTestMissing(t, filepath.Join(target, "new.go"))
}

func TestManagedOutputRejectsUnsafeManifest(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	writeOutputTestFile(t, filepath.Join(target, outputManifestName), `{"version":1,"files":[{"path":"../outside","sha256":"x"}]}`)
	output := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(output.StageDir(), "new.go"), "new")
	if err := output.Commit(); err == nil {
		t.Fatal("expected unsafe manifest error")
	}
	assertOutputTestMissing(t, filepath.Join(target, "new.go"))
}

func TestManagedOutputRejectsSymlinkedTargetParent(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "generated")
	outside := filepath.Join(root, "outside")
	writeOutputTestFile(t, filepath.Join(outside, "keep.go"), "outside")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(target, "nested")); err != nil {
		t.Skipf("create symlink: %v", err)
	}

	output := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(output.StageDir(), "nested", "keep.go"), "generated")
	if err := output.Commit(); err == nil || !strings.Contains(err.Error(), "contains symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
	assertOutputTestContent(t, filepath.Join(outside, "keep.go"), "outside")
}

func newOutputTestTransaction(t *testing.T, target string) *ManagedOutput {
	t.Helper()
	output, err := NewManagedOutput(target)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(output.Abort)
	return output
}

func writeOutputTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertOutputTestContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), expected) {
		t.Fatalf("expected %s to contain %q, got %q", path, expected, content)
	}
}

func assertOutputTestMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, err=%v", path, err)
	}
}
