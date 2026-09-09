package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunSkelcGenTS(t *testing.T) {
	dir := t.TempDir()
	tsOut := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "ts", "--api", "--skel-in", dir, "--ts-out", tsOut})
	assertGenerationResult(t, result)
	assertFileMissing(t, filepath.Join(tsOut, "package.json"))
}

func TestRunSkelcGenTSWithModule(t *testing.T) {
	dir := t.TempDir()
	tsOut := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"gen", "ts", "--api", "--skel-in", dir, "--ts-out", tsOut, "--ts-as-module", "--ts-module", "@acme/skeled-user"})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	packageJSONContent, err := os.ReadFile(filepath.Join(tsOut, "package.json"))
	if err != nil {
		t.Fatalf("read generated package.json: %v", err)
	}
	if !strings.Contains(string(packageJSONContent), `"name": "@acme/skeled-user"`) {
		t.Fatalf("expected generated package.json to use ts module: %s", string(packageJSONContent))
	}
}

func TestRunSkelcGenTSUsesFlagNamesForSharedValidationErrors(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "missing module identity",
			args:     []string{"gen", "ts", "--api", "--skel-in", dir, "--ts-out", t.TempDir(), "--ts-as-module"},
			expected: "missing flag ts-module or ts-module-scope",
		},
		{
			name:     "module requires module output",
			args:     []string{"gen", "ts", "--api", "--skel-in", dir, "--ts-out", t.TempDir(), "--ts-module", "@acme/user"},
			expected: "flag ts-module requires ts-as-module",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Run(test.args)
			assertCommandErrorMessage(t, result, test.expected)
		})
	}
}

func TestRunSkelcGenTSApi(t *testing.T) {
	dir := t.TempDir()
	tsOut := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/actor.skel", `domain demo.user

pub actor ClientActor {
    via client {}
}
`)
	writeCLIFile(t, dir+"/types.skel", `domain demo.user

pub enum UserStatus {
    ACTIVE
}

pub data User {
    status: UserStatus
}

pub data PublicUnused {
    id: int
}

data InternalUser {
    id: int
}
`)
	writeCLIFile(t, dir+"/service.skel", `domain demo.user

api service UserService {
    for ClientActor

    method getUser {
        output User
    }
}

pub service InternalService {

    method getUser {
        output User
    }
}
`)

	result := Run([]string{"gen", "ts", "--api", "--skel-in", dir, "--ts-out", tsOut})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	dataContent, err := os.ReadFile(filepath.Join(tsOut, "data.ts"))
	if err != nil {
		t.Fatalf("read generated data.ts: %v", err)
	}
	if !strings.Contains(string(dataContent), "export type PublicUnused = {") {
		t.Fatalf("expected API TypeScript to include unreachable pub data: %s", string(dataContent))
	}
	if strings.Contains(string(dataContent), "InternalUser") {
		t.Fatalf("did not expect non-pub data in generated data.ts: %s", string(dataContent))
	}
	serviceContent, err := os.ReadFile(filepath.Join(tsOut, "service.ts"))
	if err != nil {
		t.Fatalf("read generated service.ts: %v", err)
	}
	if !strings.Contains(string(serviceContent), "createUserService") {
		t.Fatalf("expected API service client in generated service.ts: %s", string(serviceContent))
	}
	if strings.Contains(string(serviceContent), "createInternalService") {
		t.Fatalf("did not expect backend service client in generated service.ts: %s", string(serviceContent))
	}
	specContent, err := os.ReadFile(filepath.Join(tsOut, "spec.ts"))
	if err != nil {
		t.Fatalf("read generated spec.ts: %v", err)
	}
	if !strings.Contains(string(specContent), "UserServiceSpec") {
		t.Fatalf("expected API service spec in generated spec.ts: %s", string(specContent))
	}
	if strings.Contains(string(specContent), "InternalServiceSpec") {
		t.Fatalf("did not expect backend service spec in generated spec.ts: %s", string(specContent))
	}
}

func TestRunSkelcGenTSRequiresApiAndRejectsPub(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, filepath.Join(dir, "domain.skel"), "domain demo.order")
	for _, flags := range [][]string{nil, {"--pub"}, {"--api", "--pub"}} {
		args := append([]string{"gen", "ts", "--skel-in", dir, "--ts-out", t.TempDir()}, flags...)
		if result := Run(args); result.ExitCode != ExitCodeError {
			t.Fatalf("accepted TS flags %v: %+v", flags, result)
		}
	}
	help := Run([]string{"gen", "ts", "--help"})
	if help.ExitCode != ExitCodeSuccess || !strings.Contains(help.Stdout, "--api") || strings.Contains(help.Stdout, "--pub") {
		t.Fatalf("unexpected TS options: %s", help.Stdout)
	}
}
