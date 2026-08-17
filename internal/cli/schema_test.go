package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	ucli "github.com/urfave/cli/v3"
	schemas "go.yorun.ai/skelc/internal/schema"
)

type _ErrorWriter struct {
	err error
}

func (w _ErrorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func TestWriteIndentedJSONUsesStableEncoding(t *testing.T) {
	var output bytes.Buffer
	cmd := &ucli.Command{Writer: &output}
	if err := writeIndentedJSON(cmd, map[string]string{"value": "<>&"}, "test value"); err != nil {
		t.Fatal(err)
	}
	if output.String() != "{\n  \"value\": \"<>&\"\n}\n" {
		t.Fatalf("unexpected JSON output:\n%s", output.String())
	}
}

func TestWriteIndentedJSONReturnsWriterErrors(t *testing.T) {
	want := errors.New("write failed")
	cmd := &ucli.Command{Writer: _ErrorWriter{err: want}}
	err := writeIndentedJSON(cmd, map[string]string{"value": "test"}, "test value")
	if !errors.Is(err, want) {
		t.Fatalf("expected writer error, got %v", err)
	}
}

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
	missingTypeError := decodeSchemaCommandError(t, missingTypeResult)
	if missingTypeResult.ExitCode != ExitCodeError || missingTypeResult.Stderr != "" ||
		missingTypeError.Code != schemas.ErrorCodeInvalidArgument || !strings.Contains(missingTypeError.Message, "expected TYPE SKEL_NAME") {
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

	missingResult := Run([]string{"schema", "get", "web", "demo.user.Missing", "--skel-in", dir})
	if missingResult.ExitCode != ExitCodeSuccess || missingResult.Stdout != "null\n" || missingResult.Stderr != "" {
		t.Fatalf("unexpected missing declaration result: %+v", missingResult)
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
		commandError := decodeSchemaCommandError(t, result)
		if result.ExitCode != ExitCodeError || result.Stderr != "" ||
			commandError.Code != schemas.ErrorCodeInvalidArgument || !strings.Contains(commandError.Message, "invalid schema declaration type") {
			t.Fatalf("expected invalid type error for %v: %+v", args, result)
		}
	}
}

func TestRunSkelcSchemaErrorsUseStdoutResult(t *testing.T) {
	invalidSource := t.TempDir()
	writeCLIFile(t, filepath.Join(invalidSource, "domain.skel"), "domain demo.user")
	writeCLIFile(t, filepath.Join(invalidSource, "data.skel"), "domain demo.user\n\ndata User { id string }")

	for _, test := range []struct {
		name string
		args []string
		code schemas.ErrorCode
	}{
		{name: "snapshot argument", args: []string{"schema", "snapshot"}, code: schemas.ErrorCodeInvalidArgument},
		{name: "diff argument", args: []string{"schema", "diff"}, code: schemas.ErrorCodeInvalidArgument},
		{name: "compilation", args: []string{"schema", "list", "--skel-in", invalidSource}, code: schemas.ErrorCodeCompilationFailed},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := Run(test.args)
			if result.ExitCode != ExitCodeError {
				t.Fatalf("expected error result: %+v", result)
			}
			commandError := decodeSchemaCommandError(t, result)
			if commandError.Code != test.code || commandError.Message == "" {
				t.Fatalf("unexpected command error: %+v", commandError)
			}
		})
	}
}

func TestRunSkelcSchemaParserErrorsUseStdoutResult(t *testing.T) {
	for _, args := range [][]string{
		{"schema", "unknown"},
		{"schema", "list", "--unknown"},
		{"--log-format", "unknown", "schema", "list"},
	} {
		result := Run(args)
		commandError := decodeSchemaCommandError(t, result)
		if result.ExitCode != ExitCodeError || result.Stderr != "" ||
			commandError.Code != schemas.ErrorCodeInvalidArgument || commandError.Message == "" {
			t.Fatalf("expected structured parser error for %v: %+v", args, result)
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
	if result.Stderr != "" || report.Compatible || report.Summary.Breaking != 1 ||
		report.Changes[0].Change != "ADDED" || report.Changes[0].Impact != "BREAKING" ||
		report.Changes[0].Code != "data.member.added" {
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

func TestRunSkelcSchemaDiffUsesGitHeadBaseline(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	for _, test := range []struct {
		name             string
		directoryInput   bool
		expectedBaseline string
	}{
		{name: "directory", directoryInput: true, expectedBaseline: "HEAD:skel/data.skel"},
		{name: "single file", expectedBaseline: "HEAD:user.skel"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := t.TempDir()
			skelIn := filepath.Join(repository, "user.skel")
			sourceFile := skelIn
			baselineSource := `domain demo.user

pub data User {
    id: string
}
`
			candidateSource := strings.Replace(baselineSource, "id: string", "id: int", 1)
			if test.directoryInput {
				skelIn = filepath.Join(repository, "skel")
				sourceFile = filepath.Join(skelIn, "data.skel")
				writeCLIFile(t, filepath.Join(skelIn, "domain.skel"), "domain demo.user")
			}
			writeCLIFile(t, sourceFile, baselineSource)
			runSchemaGitCommand(t, repository, "init", "--quiet")
			runSchemaGitCommand(t, repository, "add", "--all")
			runSchemaGitCommand(t, repository,
				"-c", "user.name=Skelc Test", "-c", "user.email=skelc@example.invalid",
				"-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "baseline",
			)
			writeCLIFile(t, sourceFile, candidateSource)

			result := Run([]string{"schema", "diff", "--skel-in", skelIn})
			if result.ExitCode != ExitCodeSuccess || result.Stderr != "" {
				t.Fatalf("unexpected Git diff result: %+v", result)
			}
			var report schemas.Report
			if err := json.Unmarshal([]byte(result.Stdout), &report); err != nil {
				t.Fatalf("decode Git diff report: %v\n%s", err, result.Stdout)
			}
			if len(report.Changes) != 1 || report.Changes[0].Code != "data.member.type.changed" {
				t.Fatalf("unexpected Git diff report: %+v", report)
			}
			if report.Changes[0].Baseline == nil || report.Changes[0].Baseline.File != test.expectedBaseline {
				t.Fatalf("unexpected Git baseline position: %+v", report.Changes[0].Baseline)
			}
		})
	}
}

func TestRunSkelcSchemaDiffRequiresBaselineWithoutGitHistory(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), "domain demo.user")

	result := Run([]string{"schema", "diff", "--skel-in", dir})
	commandError := decodeSchemaCommandError(t, result)
	if result.ExitCode != ExitCodeError || result.Stderr != "" ||
		commandError.Code != schemas.ErrorCodeGitHistoryNotFound ||
		!strings.Contains(commandError.Message, "git history not found") ||
		!strings.Contains(commandError.Message, "--baseline-skel-in") {
		t.Fatalf("expected missing Git history guidance: %+v", result)
	}
}

func TestRunSkelcSchemaDiffRemapsInvalidGitBaselinePath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	repository := t.TempDir()
	skelDir := filepath.Join(repository, "skel")
	writeCLIFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.user")
	dataPath := filepath.Join(skelDir, "data.skel")
	writeCLIFile(t, dataPath, `domain demo.user

pub data User {
    id string
}
`)
	runSchemaGitCommand(t, repository, "init", "--quiet")
	runSchemaGitCommand(t, repository, "add", "--all")
	runSchemaGitCommand(t, repository,
		"-c", "user.name=Skelc Test", "-c", "user.email=skelc@example.invalid",
		"-c", "commit.gpgsign=false", "commit", "--quiet", "-m", "invalid baseline",
	)
	writeCLIFile(t, dataPath, `domain demo.user

pub data User {
    id: string
}
`)

	result := Run([]string{"schema", "diff", "--skel-in", skelDir})
	commandError := decodeSchemaCommandError(t, result)
	if result.ExitCode != ExitCodeError || commandError.Code != schemas.ErrorCodeCommandFailed ||
		!strings.Contains(commandError.Message, "HEAD:skel/data.skel") ||
		strings.Contains(commandError.Message, "skelc-schema-baseline-") {
		t.Fatalf("expected stable Git baseline error path: %+v", result)
	}
}

func decodeSchemaCommandError(t *testing.T, result Result) *schemas.CommandError {
	t.Helper()
	commandError := new(schemas.CommandError)
	if err := json.Unmarshal([]byte(result.Stdout), commandError); err != nil {
		t.Fatalf("decode schema command error: %v\n%s", err, result.Stdout)
	}
	return commandError
}

func runSchemaGitCommand(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}
