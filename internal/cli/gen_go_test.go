package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/command"
)

func TestRunSkelcGenGo(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go", "--skel-in", dir, "--go-out", goOut})
	assertGenerationResult(t, result)
}

func TestRunSkelcGenGoWritesWarningsToJSONLLogs(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/.hidden.skel", `domain demo.user`)

	result := Run([]string{"--log-format", "jsonl", "gen", "go", "--skel-in", dir, "--go-out", goOut})
	generated := new(command.GenerationResult)
	if err := json.Unmarshal([]byte(result.Stdout), generated); err != nil {
		t.Fatal(err)
	}
	var warning struct {
		Level    string `json:"level"`
		Code     string `json:"code"`
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stderr)), &warning); err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != ExitCodeSuccess || !generated.Generated ||
		warning.Level != "warn" || warning.Code != "loader.ignored-hidden-file" || warning.Severity != "warning" {
		t.Fatalf("unexpected generation result: %+v", result)
	}
}

func TestRunSkelcGenGoClassifiesOutputFailures(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), `domain demo.user`)
	blockedParent := filepath.Join(t.TempDir(), "blocked")
	writeCLIFile(t, blockedParent, "not a directory")

	result := Run([]string{"gen", "go", "--skel-in", dir, "--go-out", filepath.Join(blockedParent, "skeled")})
	commandError := decodeCommandError(t, result)
	if result.ExitCode != ExitCodeError || commandError.Code != command.ErrorCodeCommandFailed {
		t.Fatalf("expected generation output failure: %+v", result)
	}
}

func TestRunSkelcGenGoRendersDataCloneMethodsAndGenericCallbacks(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), `domain demo.user`)
	writeCLIFile(t, filepath.Join(dir, "types.skel"), `domain demo.user

data User {
    name: string
    roles: list<string>
}

data Page<TItem> {
    items: list<TItem>
}

data Users {
    page: Page<User>
}

service UserService {
    method listUsers {
        output Page<User>
    }
}
`)

	result := Run([]string{"gen", "go", "--skel-in", dir, "--go-out", goOut})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	assertFileContains(t, filepath.Join(goOut, "data.go"),
		"func (v User) Clone() User",
		"func (v Page[TItem]) CloneBy(cloneTItem func(TItem) TItem) Page[TItem]",
		"func (v Users) Clone() Users",
		"v.Page.CloneBy(func(value User) User {")
	assertFileContains(t, filepath.Join(goOut, "service.go"),
		"CloneResult: func(value any) any {",
		"source.CloneBy(func(value User) User {")
}

func TestRunSkelcGenGoPreservesUnmanagedOutput(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, filepath.Join(goOut, ".hidden"), "hidden")
	writeCLIFile(t, filepath.Join(goOut, ".hidden-dir", "old.go"), "old")
	writeCLIFile(t, filepath.Join(goOut, "old.go"), "old")

	result := Run([]string{"gen", "go", "--skel-in", dir, "--go-out", goOut})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	assertFileContains(t, filepath.Join(goOut, ".hidden"), "hidden")
	assertFileContains(t, filepath.Join(goOut, ".hidden-dir", "old.go"), "old")
	assertFileContains(t, filepath.Join(goOut, "old.go"), "old")
	assertFileContains(t, filepath.Join(goOut, "doc.go"), "package skeled")
	assertFileContains(t, filepath.Join(goOut, "doc.go"), common.GeneratedFileMarker)
	assertFileMissing(t, filepath.Join(goOut, ".skelc-manifest.json"))
}

func TestRunSkelcGenGoDoesNotChangeOutputWhenCompilationFails(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, filepath.Join(dir, "user.skel"), `data User { id: string }`)
	writeCLIFile(t, filepath.Join(goOut, "old.go"), "old")

	result := Run([]string{"gen", "go", "--skel-in", dir, "--go-out", goOut})

	if result.ExitCode == ExitCodeSuccess {
		t.Fatal("expected generation failure")
	}
	assertFileContains(t, filepath.Join(goOut, "old.go"), "old")
}

func TestRunSkelcGenGoModule(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go-module", "--skel-in", dir, "--go-out", goOut, "--go-module-prefix", "github.com/acme/skel"})
	assertGenerationResult(t, result)
	assertFileContains(t, filepath.Join(goOut, "go.mod"), "go.yorun.ai/vine v0.13.1")
}

func TestRunSkelcGenGoModuleWithGoVineVersion(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go-module", "--skel-in", dir, "--go-out", goOut, "--go-module-prefix", "github.com/acme/skel", "--go-vine-version", "v1.2.3"})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	assertFileContains(t, filepath.Join(goOut, "go.mod"), "go.yorun.ai/vine v1.2.3")
}

func TestRunSkelcGenGoModuleRejectsLowGoVineVersion(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go-module", "--skel-in", dir, "--go-out", goOut, "--go-module-prefix", "github.com/acme/skel", "--go-vine-version", "v0.8.0"})

	assertCommandErrorMessage(t, result, "go-vine-version v0.8.0 is lower than minimum v0.13.1")
}

func TestRunSkelcGenGoModuleRejectsGoVineVersionWithoutVPrefix(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go-module", "--skel-in", dir, "--go-out", goOut, "--go-module-prefix", "github.com/acme/skel", "--go-vine-version", "1.2.3"})

	assertCommandErrorMessage(t, result, "go-vine-version 1.2.3 must be v-prefixed semantic version")
}

func TestRunSkelcGenGoModuleRejectsPubFlag(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go-module", "--pub", "--skel-in", dir, "--go-out", goOut, "--go-module-prefix", "github.com/acme/skel"})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
}

func TestRunSkelcGenGoRendersSchema(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/types.skel", `domain demo.user

pub data User {
    id: string
}
`)

	result := Run([]string{"gen", "go", "--skel-in", dir, "--go-out", goOut})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	version, err := compilerVersion()
	if err != nil {
		t.Fatal(err)
	}
	assertFileContains(t, filepath.Join(goOut, "schema.go"),
		`Domain: "demo.user"`,
		`CompilerVersion: "`+version+`"`)
}

func TestRunSkelcGenGoRejectsModuleFlags(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go", "--pub", "--skel-in", dir, "--go-out", goOut})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
}

func TestRunSkelcGenGoModuleRejectsMissingModulePrefix(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go-module", "--skel-in", dir, "--go-out", goOut})

	assertCommandErrorMessage(t, result, "missing flag go-module or go-module-prefix")
}

func TestRunSkelcGenGoModuleUsesFlagNameForSharedValidationError(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{
		"gen", "go-module", "--skel-in", dir, "--go-out", goOut,
		"--go-module-prefix", "github.com/acme/skel/",
	})

	assertCommandErrorMessage(t, result, "flag go-module-prefix must not end with /")
}

func TestRunSkelcGenGoModuleWithModulePrefix(t *testing.T) {
	dir := t.TempDir()
	goOut := filepath.Join(t.TempDir(), "skeled")
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "go-module", "--skel-in", dir, "--go-out", goOut, "--go-module-prefix", "github.com/acme/skel"})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcGenGoModuleWithSkelImportAndGoImport(t *testing.T) {
	root := t.TempDir()
	userDir := filepath.Join(root, "user")
	bookerDir := filepath.Join(root, "booker")
	goOut := filepath.Join(root, "booker_gen")
	writeCLIFile(t, filepath.Join(userDir, "domain.skel"), `domain user`)
	writeCLIFile(t, filepath.Join(userDir, "types.skel"), `domain user

pub data UserSummary {
    userId: int
}
`)
	writeCLIFile(t, filepath.Join(bookerDir, "domain.skel"), `domain booker

import user as account
`)
	writeCLIFile(t, filepath.Join(bookerDir, "types.skel"), `domain booker

pub data LoanRecord {
    borrower: account.UserSummary
}
`)

	result := Run([]string{
		"gen", "go-module",
		"--skel-in", bookerDir,
		"--go-out", goOut,
		"--skel-import", "user=" + userDir,
		"--go-import", "user=go.yorun.ai/app/vine/demo/userpub",
		"--go-module", "go.yorun.ai/app/vine/demo/bookerpub",
	})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	dataContent, err := os.ReadFile(filepath.Join(goOut, "data.go"))
	if err != nil {
		t.Fatalf("read generated data.go: %v", err)
	}
	if !strings.Contains(string(dataContent), `account "go.yorun.ai/app/vine/demo/userpub"`) {
		t.Fatalf("expected user import in generated data.go: %s", string(dataContent))
	}
	if !strings.Contains(string(dataContent), "Borrower account.UserSummary") {
		t.Fatalf("expected external user type in generated data.go: %s", string(dataContent))
	}
	goModContent, err := os.ReadFile(filepath.Join(goOut, "go.mod"))
	if err != nil {
		t.Fatalf("read generated go.mod: %v", err)
	}
	if !strings.Contains(string(goModContent), "module go.yorun.ai/app/vine/demo/bookerpub") {
		t.Fatalf("unexpected go.mod content: %s", string(goModContent))
	}
}

func TestRunSkelcGenGoModuleWithDefaultSkelImportUsesPubPackageName(t *testing.T) {
	root := t.TempDir()
	userDir := filepath.Join(root, "user")
	bookerDir := filepath.Join(root, "booker")
	goOut := filepath.Join(root, "booker_gen")
	writeCLIFile(t, filepath.Join(userDir, "domain.skel"), `domain user`)
	writeCLIFile(t, filepath.Join(userDir, "types.skel"), `domain user

pub data UserSummary {
    userId: int
}
`)
	writeCLIFile(t, filepath.Join(bookerDir, "domain.skel"), `domain booker

import user
`)
	writeCLIFile(t, filepath.Join(bookerDir, "types.skel"), `domain booker

pub data LoanRecord {
    borrower: user.UserSummary
}
`)

	result := Run([]string{
		"gen", "go-module",
		"--skel-in", bookerDir,
		"--go-out", goOut,
		"--skel-import", "user=" + userDir,
		"--go-module-prefix", "go.yorun.ai/app/vine/demo",
	})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	dataContent, err := os.ReadFile(filepath.Join(goOut, "data.go"))
	if err != nil {
		t.Fatalf("read generated data.go: %v", err)
	}
	if !strings.Contains(string(dataContent), `"go.yorun.ai/app/vine/demo/userpub"`) {
		t.Fatalf("expected user import in generated data.go: %s", string(dataContent))
	}
	if strings.Contains(string(dataContent), `user "go.yorun.ai/app/vine/demo/userpub"`) {
		t.Fatalf("expected user import without alias in generated data.go: %s", string(dataContent))
	}
	if !strings.Contains(string(dataContent), "Borrower userpub.UserSummary") {
		t.Fatalf("expected external userpub type in generated data.go: %s", string(dataContent))
	}
}
