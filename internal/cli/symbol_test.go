package cli

import (
	"strings"
	"testing"
)

func TestRunSkelcSymbolList(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/symbols.skel", `domain demo.user

import app

pub actor ClientActor {
    via client {}
}

enum UserStatus {
    ACTIVE
}

pub data User {
    id: string
    context: app.AppContext
}

config SiteConfig eternal {
    title: string
}

event UserCreatedEvent {
    payload {
        userId: string
    }
}

pub service UserService {
    for ClientActor

    method getUser {
        output User
    }
}

web UserPortalWeb {
    for ClientActor
}

task RebuildUserIndexTask {
    trigger atTime {
        input {
            startAt: localdatetime
        }
    }
}
`)

	result := Run([]string{"symbol", "list", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	expected := strings.Join([]string{
		"pub  actor    demo.user.ClientActor",
		"---  config   demo.user.SiteConfig",
		"pub  data     demo.user.User",
		"---  enum     demo.user.UserStatus",
		"---  event    demo.user.UserCreatedEvent",
		"pub  service  demo.user.UserService",
		"---  task     demo.user.RebuildUserIndexTask",
		"---  web      demo.user.UserPortalWeb",
	}, "\n") + "\n"
	if result.Stdout != expected {
		t.Fatalf("unexpected stdout:\n%s", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcSymbolListDoesNotRequireSkelImport(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.booker`)
	writeCLIFile(t, dir+"/symbols.skel", `domain demo.booker

import user

data Loan {
    borrower: user.User
}
`)

	result := Run([]string{"symbol", "list", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	if result.Stdout != "---  data  demo.booker.Loan\n" {
		t.Fatalf("unexpected stdout:\n%s", result.Stdout)
	}
}

func TestRunSkelcSymbolListJSON(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/symbols.skel", `domain demo.user

pub actor ClientActor {
    via client {}
}

pub data User {
    id: string
}

data InternalUser {
    id: string
}
`)

	result := Run([]string{"symbol", "list", "--output-format", "json", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	expected := `[
  {
    "pub": true,
    "name": "ClientActor",
    "type": "actor",
    "skelName": "demo.user.ClientActor"
  },
  {
    "pub": false,
    "name": "InternalUser",
    "type": "data",
    "skelName": "demo.user.InternalUser"
  },
  {
    "pub": true,
    "name": "User",
    "type": "data",
    "skelName": "demo.user.User"
  }
]
`
	if result.Stdout != expected {
		t.Fatalf("unexpected stdout:\n%s", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcSymbolGet(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/symbols.skel", `domain demo.user

actor ClientActor {
    via client {}
}

pub data User {
    id: string
}
`)

	result := Run([]string{"symbol", "get", "demo.user.User", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	if result.Stdout != "pub  data  demo.user.User\n" {
		t.Fatalf("unexpected stdout:\n%s", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcSymbolGetJSON(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)
	writeCLIFile(t, dir+"/symbols.skel", `domain demo.user

pub data User {
    id: string
}
`)

	result := Run([]string{"symbol", "get", "demo.user.User", "--output-format", "json", "--skel-in", dir})

	if result.ExitCode != ExitCodeSuccess {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	expected := `{
  "pub": true,
  "name": "User",
  "type": "data",
  "skelName": "demo.user.User"
}
`
	if result.Stdout != expected {
		t.Fatalf("unexpected stdout:\n%s", result.Stdout)
	}
	if result.Stderr != "" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}

func TestRunSkelcSymbolGetMissing(t *testing.T) {
	dir := t.TempDir()
	writeCLIFile(t, dir+"/domain.skel", `domain demo.user`)

	result := Run([]string{"symbol", "get", "demo.user.Missing", "--skel-in", dir})

	if result.ExitCode != ExitCodeError {
		t.Fatalf("unexpected exit code: %d, stderr=%q", result.ExitCode, result.Stderr)
	}
	if result.Stdout != "" {
		t.Fatalf("unexpected stdout:\n%s", result.Stdout)
	}
	if result.Stderr != "Error: symbol not found: demo.user.Missing" {
		t.Fatalf("unexpected stderr: %q", result.Stderr)
	}
}
