package cli

import (
	"encoding/json"
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
	var entries []*schemas.Entry
	if err := json.Unmarshal([]byte(listResult.Stdout), &entries); err != nil {
		t.Fatalf("decode list result: %v\n%s", err, listResult.Stdout)
	}
	if len(entries) != 2 || entries[0].Kind != "data" || entries[1].Kind != "resource" {
		t.Fatalf("unexpected list entries: %+v", entries)
	}
	filteredResult := Run([]string{"schema", "list", "data", "--skel-in", dir})
	if filteredResult.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected filtered list result: %+v", filteredResult)
	}
	entries = nil
	if err := json.Unmarshal([]byte(filteredResult.Stdout), &entries); err != nil || len(entries) != 1 || entries[0].Kind != "data" {
		t.Fatalf("unexpected filtered list entries: %+v, err=%v", entries, err)
	}

	missingTypeResult := Run([]string{"schema", "get", "demo.user.User", "--skel-in", dir})
	if missingTypeResult.ExitCode != ExitCodeError || !strings.Contains(missingTypeResult.Stderr, "expected TYPE SKEL_NAME") {
		t.Fatalf("expected missing type error: %+v", missingTypeResult)
	}

	getResult := Run([]string{"schema", "get", "data", "demo.user.User", "--skel-in", dir})
	if getResult.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected get result: %+v", getResult)
	}
	var declaration schemas.Declaration
	if err := json.Unmarshal([]byte(getResult.Stdout), &declaration); err != nil {
		t.Fatalf("decode declaration: %v\n%s", err, getResult.Stdout)
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

func TestRunSkelcSchemaSnapshotFullDocument(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(dir, "data.skel"), `domain demo.user

pub data User {
    id: string
}

data InternalUser {
    id: string
}
`)

	result := Run([]string{"schema", "snapshot", "--skel-in", dir})
	if result.ExitCode != ExitCodeSuccess || result.Stdout == "" || result.Stderr != "" {
		t.Fatalf("unexpected snapshot result: %+v", result)
	}
	document, err := schemas.Decode(strings.NewReader(result.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Declarations) != 2 || schemas.Find(document, "data", "demo.user.User") == nil {
		t.Fatalf("unexpected schema: %+v", document)
	}
	internal := schemas.Find(document, "data", "demo.user.InternalUser")
	if internal == nil || internal.Pub {
		t.Fatalf("expected private declaration in full schema: %+v", internal)
	}
}

func TestRunSkelcSchemaSnapshotKeepsImportsOpaque(t *testing.T) {
	source := t.TempDir()
	aliasSource := t.TempDir()
	writeCLIFile(t, filepath.Join(source, "domain.skel"), `domain demo.order`)
	writeCLIFile(t, filepath.Join(aliasSource, "domain.skel"), `domain demo.order`)
	sourceSchema := `domain demo.order

import demo.user as identity

pub data Order {
    owner: identity.User
}

pub service OrderService {
    require identity.User:read

    method get {
        output Order
    }
}
`
	writeCLIFile(t, filepath.Join(source, "schema.skel"), sourceSchema)
	writeCLIFile(t, filepath.Join(aliasSource, "schema.skel"), strings.ReplaceAll(sourceSchema, "identity", "account"))

	shallow := Run([]string{"schema", "snapshot", "--skel-in", source})
	if shallow.ExitCode != ExitCodeSuccess {
		t.Fatalf("export with opaque imports failed: %+v", shallow)
	}
	aliasSnapshot := Run([]string{"schema", "snapshot", "--skel-in", aliasSource})
	if aliasSnapshot.ExitCode != ExitCodeSuccess {
		t.Fatalf("snapshot with alternate import alias failed: %+v", aliasSnapshot)
	}
	if shallow.Stdout != aliasSnapshot.Stdout {
		t.Fatalf("import alias changed schema output:\nidentity alias:\n%s\naccount alias:\n%s", shallow.Stdout, aliasSnapshot.Stdout)
	}
	document, err := schemas.Decode(strings.NewReader(shallow.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	order := schemas.Find(document, "data", "demo.order.Order")
	if order == nil || order.Data.Members[0].Type.Kind != "importedReference" || order.Data.Members[0].Type.Name != "demo.user.User" {
		t.Fatalf("unexpected imported data reference: %+v", order)
	}
	service := schemas.Find(document, "service", "demo.order.OrderService")
	if service == nil || service.Service.Require == nil || service.Service.Require.Mode != "reference" ||
		service.Service.Require.Check.Resource != "demo.user.User" {
		t.Fatalf("unexpected imported permission reference: %+v", service)
	}
	diff := Run([]string{
		"schema", "diff", "--baseline-skel-in", source, "--skel-in", aliasSource,
	})
	var diffReport schemas.Report
	if err := json.Unmarshal([]byte(diff.Stdout), &diffReport); err != nil {
		t.Fatalf("decode diff report: %v\n%s", err, diff.Stdout)
	}
	if diff.ExitCode != ExitCodeSuccess || !diffReport.Compatible || len(diffReport.Changes) != 0 {
		t.Fatalf("source diff should not require imports or observe aliases: %+v", diff)
	}
}

func TestRunSkelcSchemaDiffOutputsCompleteReport(t *testing.T) {
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

	result := Run([]string{"schema", "diff", "--baseline-skel-in", baseline, "--skel-in", candidate})
	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("expected completed diff, got %+v", result)
	}
	var report schemas.Report
	if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
		t.Fatalf("decode diff report: %v\n%s", err, result.Stdout)
	}
	if result.Stderr != "" || report.Compatible || report.Summary.Breaking != 1 || report.Changes[0].Code != "data.member.added" {
		t.Fatalf("unexpected diff output: %+v", result)
	}
}

func TestRunSkelcSchemaDiffChecksPrivateDeclarations(t *testing.T) {
	baseline := t.TempDir()
	candidate := t.TempDir()
	writeCLIFile(t, filepath.Join(baseline, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(baseline, "data.skel"), `domain demo.user

data InternalUser {
    id: string
}
`)
	writeCLIFile(t, filepath.Join(candidate, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(candidate, "data.skel"), `domain demo.user

data InternalUser {
    id: string
    name: string
}
`)

	result := Run([]string{
		"schema", "diff", "--baseline-skel-in", baseline, "--skel-in", candidate,
	})
	if result.ExitCode != ExitCodeSuccess || !strings.Contains(result.Stdout, "data.member.added") {
		t.Fatalf("expected private declaration incompatibility: %+v", result)
	}
}

func TestRunSkelcSchemaDiffReportsDangerousSourceChange(t *testing.T) {
	baseline := t.TempDir()
	candidate := t.TempDir()
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

	result := Run([]string{
		"schema", "diff", "--baseline-skel-in", baseline, "--skel-in", candidate,
	})
	if result.ExitCode != ExitCodeSuccess || result.Stderr != "" {
		t.Fatalf("unexpected diff result: %+v", result)
	}
	var report schemas.Report
	if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
		t.Fatalf("decode report: %v\n%s", err, result.Stdout)
	}
	if !report.Compatible || report.Summary.Dangerous != 1 || report.Changes[0].Code != "enum.item.added" {
		t.Fatalf("unexpected report: %+v", report)
	}
}
