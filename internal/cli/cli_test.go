package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/command"
)

func decodeCommandError(t *testing.T, result Result) *command.Error {
	t.Helper()
	commandError := new(command.Error)
	if err := json.Unmarshal([]byte(result.Stdout), commandError); err != nil {
		t.Fatalf("decode command error: %v\n%s", err, result.Stdout)
	}
	return commandError
}

func TestRunUnknownCommandReturnsJSONError(t *testing.T) {
	result := Run([]string{"unknown"})

	commandError := decodeCommandError(t, result)
	if result.ExitCode != ExitCodeError || result.Stderr != "" ||
		commandError.Code != command.ErrorCodeInvalidArgument ||
		!strings.Contains(commandError.Message, "No help topic for 'unknown'") {
		t.Fatalf("unexpected command error: %+v", result)
	}
}

func assertCommandErrorMessage(t *testing.T, result Result, message string) {
	t.Helper()
	commandError := decodeCommandError(t, result)
	if result.ExitCode != ExitCodeError || result.Stderr != "" ||
		commandError.Code != command.ErrorCodeInvalidArgument || commandError.Message != message {
		t.Fatalf("unexpected command error: %+v", result)
	}
}

func assertGenerationResult(t *testing.T, result Result) {
	t.Helper()
	generated := new(command.GenerationResult)
	if err := json.Unmarshal([]byte(result.Stdout), generated); err != nil {
		t.Fatalf("decode generation result: %v\n%s", err, result.Stdout)
	}
	if result.ExitCode != ExitCodeSuccess || !generated.Generated || result.Stderr != "" {
		t.Fatalf("unexpected generation result: %+v", result)
	}
}

func writeCLIFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertFileContains(t *testing.T, path string, expected ...string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(content)
	for _, item := range expected {
		if !strings.Contains(text, item) {
			t.Fatalf("expected %s to contain %q:\n%s", path, item, text)
		}
	}
}

func assertFileMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, err=%v", path, err)
	}
}

func runInDir(t *testing.T, dir string, fn func()) {
	t.Helper()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("restore cwd %s: %v", originalDir, err)
		}
	}()
	fn()
}

func canonicalTestDir(t *testing.T, dir string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("canonicalize %s: %v", dir, err)
	}
	return canonical
}
