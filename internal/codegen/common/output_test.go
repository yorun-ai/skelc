package common

import (
	"errors"
	"io/fs"
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

func TestManagedOutputRollsBackPartialCommit(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	initial := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(initial.StageDir(), "a.go"), "old a")
	writeOutputTestFile(t, filepath.Join(initial.StageDir(), "b.go"), "old b")
	if err := initial.Commit(); err != nil {
		t.Fatal(err)
	}
	manifestBefore, err := os.ReadFile(filepath.Join(target, outputManifestName))
	if err != nil {
		t.Fatal(err)
	}

	output := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(output.StageDir(), "a.go"), "new a")
	writeOutputTestFile(t, filepath.Join(output.StageDir(), "b.go"), "new b")
	writes := 0
	output.writeFile = func(path string, content []byte, mode fs.FileMode) error {
		writes++
		if writes == 2 {
			return errors.New("injected write failure")
		}
		return atomicWriteFile(path, content, mode)
	}
	if err := output.Commit(); err == nil || !strings.Contains(err.Error(), "injected write failure") {
		t.Fatalf("expected injected commit failure, got %v", err)
	}

	assertOutputTestContent(t, filepath.Join(target, "a.go"), "old a")
	assertOutputTestContent(t, filepath.Join(target, "b.go"), "old b")
	assertOutputTestExact(t, filepath.Join(target, outputManifestName), manifestBefore)
}

func TestManagedOutputRollsBackNewTarget(t *testing.T) {
	target := filepath.Join(t.TempDir(), "generated")
	output := newOutputTestTransaction(t, target)
	writeOutputTestFile(t, filepath.Join(output.StageDir(), "a.go"), "new a")
	writeOutputTestFile(t, filepath.Join(output.StageDir(), "b.go"), "new b")
	writes := 0
	output.writeFile = func(path string, content []byte, mode fs.FileMode) error {
		writes++
		if writes == 2 {
			return errors.New("injected write failure")
		}
		return atomicWriteFile(path, content, mode)
	}
	if err := output.Commit(); err == nil {
		t.Fatal("expected commit failure")
	}
	assertOutputTestMissing(t, target)
}

func TestCommitManagedOutputsRollsBackEarlierTargets(t *testing.T) {
	root := t.TempDir()
	firstTarget := filepath.Join(root, "regular")
	secondTarget := filepath.Join(root, "public")
	manifests := map[string][]byte{}
	for _, target := range []string{firstTarget, secondTarget} {
		initial := newOutputTestTransaction(t, target)
		writeOutputTestFile(t, filepath.Join(initial.StageDir(), "generated.go"), "old")
		if err := initial.Commit(); err != nil {
			t.Fatal(err)
		}
		manifest, err := os.ReadFile(filepath.Join(target, outputManifestName))
		if err != nil {
			t.Fatal(err)
		}
		manifests[target] = manifest
	}

	first := newOutputTestTransaction(t, firstTarget)
	second := newOutputTestTransaction(t, secondTarget)
	writeOutputTestFile(t, filepath.Join(first.StageDir(), "generated.go"), "new regular")
	writeOutputTestFile(t, filepath.Join(second.StageDir(), "generated.go"), "new public")
	second.writeFile = func(string, []byte, fs.FileMode) error {
		return errors.New("injected second-output failure")
	}
	if err := CommitManagedOutputs([]*ManagedOutput{first, second}); err == nil || !strings.Contains(err.Error(), "injected second-output failure") {
		t.Fatalf("expected second output failure, got %v", err)
	}

	assertOutputTestContent(t, filepath.Join(firstTarget, "generated.go"), "old")
	assertOutputTestContent(t, filepath.Join(secondTarget, "generated.go"), "old")
	assertOutputTestExact(t, filepath.Join(firstTarget, outputManifestName), manifests[firstTarget])
	assertOutputTestExact(t, filepath.Join(secondTarget, outputManifestName), manifests[secondTarget])
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

func assertOutputTestExact(t *testing.T, path string, expected []byte) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(expected) {
		t.Fatalf("expected %s to contain %q, got %q", path, expected, content)
	}
}
