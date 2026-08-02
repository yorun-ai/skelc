package fileutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceAllRollsBackPartialCommit(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.skel")
	secondPath := filepath.Join(dir, "second.skel")
	writeTestFile(t, firstPath, "old first")
	writeTestFile(t, secondPath, "old second")

	writes := 0
	err := replaceAll([]Replacement{
		{Path: firstPath, Content: []byte("new first"), Mode: 0o644},
		{Path: secondPath, Content: []byte("new second"), Mode: 0o644},
	}, func(source string, target string) error {
		writes++
		if writes == 2 {
			return errors.New("injected commit failure")
		}
		return os.Rename(source, target)
	})
	if err == nil || !strings.Contains(err.Error(), "injected commit failure") {
		t.Fatalf("expected injected commit failure, got %v", err)
	}
	assertTestFile(t, firstPath, "old first")
	assertTestFile(t, secondPath, "old second")
	if temporary, err := filepath.Glob(filepath.Join(dir, ".skelc-file-*")); err != nil || len(temporary) != 0 {
		t.Fatalf("unexpected transaction files after rollback: %q, err=%v", temporary, err)
	}
}

func TestReplaceAllStagesEveryFileBeforeCommit(t *testing.T) {
	dir := t.TempDir()
	firstPath := filepath.Join(dir, "first.skel")
	writeTestFile(t, firstPath, "old first")

	err := ReplaceAll([]Replacement{
		{Path: firstPath, Content: []byte("new first"), Mode: 0o644},
		{Path: filepath.Join(dir, "missing", "second.skel"), Content: []byte("new second"), Mode: 0o644},
	})
	if err == nil {
		t.Fatal("expected staging failure")
	}
	assertTestFile(t, firstPath, "old first")
}

func TestReplaceAllRemovesNewFilesOnRollback(t *testing.T) {
	dir := t.TempDir()
	newPath := filepath.Join(dir, "new.skel")
	existingPath := filepath.Join(dir, "existing.skel")
	writeTestFile(t, existingPath, "old existing")

	writes := 0
	err := replaceAll([]Replacement{
		{Path: newPath, Content: []byte("new"), Mode: 0o644},
		{Path: existingPath, Content: []byte("new existing"), Mode: 0o644},
	}, func(source string, target string) error {
		writes++
		if writes == 2 {
			return errors.New("injected commit failure")
		}
		return os.Rename(source, target)
	})
	if err == nil {
		t.Fatal("expected injected commit failure")
	}
	if _, err := os.Stat(newPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected new file to be removed, got %v", err)
	}
	assertTestFile(t, existingPath, "old existing")
}

func TestReplaceAllRollsBackWhenDirectorySyncFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "domain.skel")
	writeTestFile(t, path, "old")
	originalSync := syncReplacementDirectory
	t.Cleanup(func() { syncReplacementDirectory = originalSync })
	calls := 0
	syncReplacementDirectory = func(string) error {
		calls++
		if calls == 1 {
			return errors.New("injected directory sync failure")
		}
		return nil
	}

	err := ReplaceAll([]Replacement{{Path: path, Content: []byte("new"), Mode: 0o644}})
	if err == nil || !strings.Contains(err.Error(), "injected directory sync failure") {
		t.Fatalf("expected directory sync failure, got %v", err)
	}
	assertTestFile(t, path, "old")
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertTestFile(t *testing.T, path string, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != expected {
		t.Fatalf("unexpected content for %s: %q", path, content)
	}
}
