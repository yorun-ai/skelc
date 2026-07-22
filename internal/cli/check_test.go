package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/parser"
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

func TestRunSkelcCheckWritesTextLoaderWarnings(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/.hidden.skel", `domain demo.user`)

	result := Run([]string{"check", "--skel-in", dir})
	if result.ExitCode != ExitCodeSuccess || !strings.Contains(result.Stdout, ".hidden.skel ignored (HIDDEN_FILE)") {
		t.Fatalf("unexpected result: %+v", result)
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

func TestRunSkelcCheckWritesMultipleStructuredSyntaxDiagnostics(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", "domain demo.user")
	writeCLIFile(t, dir+"/types.skel", `domain demo.user
data User {
    first string
    second:
}
`)

	result := Run([]string{"check", "--log-format", "jsonl", "--skel-in", dir})
	if result.ExitCode != ExitCodeError {
		t.Fatalf("expected check failure: %+v", result)
	}
	lines := strings.Split(strings.TrimSpace(result.Stderr), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two diagnostics, got %d: %s", len(lines), result.Stderr)
	}
	for _, line := range lines {
		entry := &_LogEntry{}
		if err := json.Unmarshal([]byte(line), entry); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(entry.Code, "syntax.") || entry.Severity != parser.DiagnosticSeverityError {
			t.Fatalf("unexpected structured diagnostic: %+v", entry)
		}
		if entry.Range.End.Column <= entry.Range.Start.Column {
			t.Fatalf("expected non-empty diagnostic range: %+v", entry.Range)
		}
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

func TestRunSkelcCheckStillValidatesLocalDeclarationsWithMissingImports(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.booker`)
	writeCLIFile(t, dir+"/types.skel", `domain demo.booker
import demo.user as user

data invalidName {
    owner: user.User
}
`)

	result := Run([]string{"check", "--skel-in", dir})

	if result.ExitCode != ExitCodeError || !strings.Contains(result.Stderr, "Data") {
		t.Fatalf("expected local naming diagnostic, got exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}
	if strings.Contains(result.Stderr, "skel import") {
		t.Fatalf("check must continue allowing unresolved imports, got stderr=%q", result.Stderr)
	}
}

func TestRunSkelcCheckReportsMultipleSemanticErrors(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/types.skel", `domain demo.user
data User { missing: MissingUser }
data Order { missing: MissingOrder }
`)

	result := Run([]string{"check", "--skel-in", dir})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("expected error exit code, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Stderr, "definition of MissingUser not found") ||
		!strings.Contains(result.Stderr, "definition of MissingOrder not found") {
		t.Fatalf("expected both semantic errors, got stderr=%q", result.Stderr)
	}
	if strings.Count(result.Stderr, "Error: ") != 2 {
		t.Fatalf("expected two text diagnostics, got stderr=%q", result.Stderr)
	}
}

func TestRunSkelcCheckReportsMultipleJSONLErrors(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/types.skel", `domain demo.user
data User { missing: MissingUser }
data Order { missing: MissingOrder }
`)

	result := Run([]string{"--log-format", "jsonl", "check", "--skel-in", dir})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("expected error exit code, got %d", result.ExitCode)
	}
	lines := strings.Split(strings.TrimSpace(result.Stderr), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected two JSONL diagnostics, got stderr=%q", result.Stderr)
	}
	for _, line := range lines {
		entry := &_LogEntry{}
		if err := json.Unmarshal([]byte(line), entry); err != nil {
			t.Fatalf("parse stderr jsonl: %v\n%s", err, result.Stderr)
		}
		if entry.Level != logLevelError {
			t.Fatalf("unexpected level: %q", entry.Level)
		}
	}
}

func TestRunSkelcCheckReportsSyntaxErrorsFromMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/user.skel", "domain demo.user\ndata User {")
	writeCLIFile(t, dir+"/order.skel", "domain demo.user\ndata Order {")

	result := Run([]string{"check", "--skel-in", dir})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("expected error exit code, got %d", result.ExitCode)
	}
	if strings.Count(result.Stderr, "Error: ") != 2 {
		t.Fatalf("expected two syntax diagnostics, got stderr=%q", result.Stderr)
	}
	if !strings.Contains(result.Stderr, "user.skel") || !strings.Contains(result.Stderr, "order.skel") {
		t.Fatalf("expected both source paths, got stderr=%q", result.Stderr)
	}
}

func TestRunSkelcCheckPreservesDomainFileRestrictions(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", "domain demo.user\ndata User {}\n")

	result := Run([]string{"check", "--skel-in", dir})

	if result.ExitCode != ExitCodeError || !strings.Contains(result.Stderr, "can only contain domain declaration and @desc") {
		t.Fatalf("expected domain file restriction, got exit=%d stderr=%q", result.ExitCode, result.Stderr)
	}
}
