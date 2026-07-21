package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRunSkelcCheck(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"check", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	if result.Stdout != "" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcCheckRejectsAllow(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/service.skel", `domain demo.user

actor ClientActor {
    via client {}
}

service UserService {
    allow ClientActor

    method ping {}
}`)

	result := Run([]string{"check", "--skel-in", dir})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("expected error exit code, got %d", result.ExitCode)
	}
	if result.Stdout != "" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
	if !strings.Contains(result.Stderr, `unexpected token "allow"`) {
		t.Fatalf("expected syntax error for allow, got stderr=%q", result.Stderr)
	}
}

func TestRunSkelcCheckWritesJSONLLoaderWarnings(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/.hidden.skel", `domain demo.user`)

	result := Run([]string{"check", "--log-format", "jsonl", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	entry := &_LogEntry{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stdout)), entry); err != nil {
		t.Fatalf("parse stdout jsonl: %v\n%s", err, result.Stdout)
	}
	if entry.Level != logLevelWarn {
		t.Fatalf("unexpected level: %q", entry.Level)
	}
	if !strings.Contains(entry.Message, ".hidden.skel ignored (HIDDEN_FILE)") {
		t.Fatalf("unexpected message: %q", entry.Message)
	}
}

func TestRunSkelcCheckWritesJSONLErrors(t *testing.T) {
	result := Run([]string{"--log-format", "jsonl", "check"})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	entry := &_LogEntry{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stderr)), entry); err != nil {
		t.Fatalf("parse stderr jsonl: %v\n%s", err, result.Stderr)
	}
	if entry.Level != logLevelError {
		t.Fatalf("unexpected level: %q", entry.Level)
	}
	if entry.Message != "missing flag skel-in" {
		t.Fatalf("unexpected message: %q", entry.Message)
	}
}

func TestRunSkelcCheckDoesNotRequireSkelImport(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.booker`)
	writeCLIFile(t, dir+"/types.skel", `domain demo.booker
import demo.user as user

data Booking {
    owner: user.User
}`)

	result := Run([]string{"check", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	if result.Stdout != "" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}
