package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	schemas "go.yorun.ai/skelc/internal/schema"
)

func TestRunSkelcSchemaListAndGet(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(dir, "schema.skel"), `domain demo.user

pub data User {
    id: string
}

pub resource User {
    action view
}
`)

	listResult := Run([]string{"schema", "list", "--skel-in", dir})
	if listResult.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected list result: %+v", listResult)
	}
	if listResult.Stdout != "pub  data      demo.user.User\npub  resource  demo.user.User\n" {
		t.Fatalf("unexpected list output:\n%s", listResult.Stdout)
	}
	filteredResult := Run([]string{"schema", "list", "data", "--skel-in", dir})
	if filteredResult.ExitCode != ExitCodeSuccess || filteredResult.Stdout != "pub  data  demo.user.User\n" {
		t.Fatalf("unexpected filtered list result: %+v", filteredResult)
	}

	missingTypeResult := Run([]string{"schema", "get", "demo.user.User", "--skel-in", dir})
	if missingTypeResult.ExitCode != ExitCodeError || !strings.Contains(missingTypeResult.Stderr, "expected TYPE SKEL_NAME") {
		t.Fatalf("expected missing type error: %+v", missingTypeResult)
	}

	getResult := Run([]string{"schema", "get", "data", "demo.user.User", "--skel-in", dir})
	if getResult.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected get result: %+v", getResult)
	}
	if getResult.Stdout != "pub data demo.user.User\n  name: User\n  members:\n    - id: string\n" {
		t.Fatalf("unexpected get text output:\n%s", getResult.Stdout)
	}

	getJSONResult := Run([]string{"schema", "get", "data", "demo.user.User", "--output-format", "json", "--skel-in", dir})
	if getJSONResult.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected get JSON result: %+v", getJSONResult)
	}
	var declaration schemas.Declaration
	if err := json.Unmarshal([]byte(getJSONResult.Stdout), &declaration); err != nil {
		t.Fatalf("decode declaration: %v\n%s", err, getJSONResult.Stdout)
	}
	if declaration.Data == nil || len(declaration.Data.Members) != 1 || declaration.Data.Members[0].Name != "id" {
		t.Fatalf("unexpected declaration: %+v", declaration)
	}
}

func TestRunSkelcSchemaQueryRejectsInvalidType(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), `domain demo.user`)

	for _, args := range [][]string{
		{"schema", "list", "unknown", "--skel-in", dir},
		{"schema", "get", "unknown", "demo.user.User", "--skel-in", dir},
	} {
		result := Run(args)
		if result.ExitCode != ExitCodeError || !strings.Contains(result.Stderr, "invalid schema declaration type") {
			t.Fatalf("expected invalid type error for %v: %+v", args, result)
		}
	}
}

func TestRunSkelcSchemaExportPublicDocument(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(t.TempDir(), "nested", "user.schema.json")
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(dir, "data.skel"), `domain demo.user

pub data User {
    id: string
}

data InternalUser {
    id: string
}
`)

	result := Run([]string{"schema", "export", "--skel-in", dir, "--schema-out", output})
	if result.ExitCode != ExitCodeSuccess || result.Stdout != "" || result.Stderr != "" {
		t.Fatalf("unexpected export result: %+v", result)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	document, err := schemas.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if document.Scope != schemas.ScopePublic || len(document.Declarations) != 1 || document.Declarations[0].SkelName != "demo.user.User" {
		t.Fatalf("unexpected schema: %+v", document)
	}
}

func TestRunSkelcSchemaCompareReturnsDedicatedExitCode(t *testing.T) {
	baseline := t.TempDir()
	candidate := t.TempDir()
	writeCLIFile(t, filepath.Join(baseline, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(baseline, "data.skel"), `domain demo.user

pub data User {
    id: string
}
`)
	writeCLIFile(t, filepath.Join(candidate, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(candidate, "data.skel"), `domain demo.user

pub data User {
    id: string
    name: string
}
`)

	result := Run([]string{"schema", "compare", "--against-skel-in", baseline, "--skel-in", candidate})
	if result.ExitCode != ExitCodeIncompatible {
		t.Fatalf("expected incompatibility exit code, got %+v", result)
	}
	if result.Stderr != "" || !strings.Contains(result.Stdout, "data.member.added") || !strings.Contains(result.Stdout, "incompatible: 1 breaking") {
		t.Fatalf("unexpected comparison output: %+v", result)
	}

	allowed := Run([]string{
		"schema", "compare", "--against-skel-in", baseline, "--skel-in", candidate, "--fail-on", "none",
	})
	if allowed.ExitCode != ExitCodeSuccess {
		t.Fatalf("fail-on none should succeed: %+v", allowed)
	}
}

func TestRunSkelcSchemaCompareSnapshotToSourceJSON(t *testing.T) {
	baseline := t.TempDir()
	candidate := t.TempDir()
	snapshot := filepath.Join(t.TempDir(), "baseline.schema.json")
	writeCLIFile(t, filepath.Join(baseline, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(baseline, "data.skel"), `domain demo.user

pub enum UserStatus {
    ACTIVE
}
`)
	writeCLIFile(t, filepath.Join(candidate, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(candidate, "data.skel"), `domain demo.user

pub enum UserStatus {
    ACTIVE
    DISABLED
}
`)

	exportResult := Run([]string{"schema", "export", "--skel-in", baseline, "--schema-out", snapshot})
	if exportResult.ExitCode != ExitCodeSuccess {
		t.Fatalf("export failed: %+v", exportResult)
	}
	result := Run([]string{
		"schema", "compare", "--against", snapshot, "--skel-in", candidate,
		"--output-format", "json", "--fail-on", "dangerous",
	})
	if result.ExitCode != ExitCodeIncompatible || result.Stderr != "" {
		t.Fatalf("unexpected comparison result: %+v", result)
	}
	var report schemas.Report
	if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, result.Stdout)
	}
	if !report.Compatible || report.Summary.Dangerous != 1 || report.Changes[0].Code != "enum.item.added" {
		t.Fatalf("unexpected report: %+v", report)
	}
}
