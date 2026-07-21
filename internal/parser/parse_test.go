package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDirectory(t *testing.T) {
	skelDir := t.TempDir()
	writeParseFile(t, filepath.Join(skelDir, "domain.skel"), "@desc(\"User domain\")\ndomain demo.user\n")
	writeParseFile(t, filepath.Join(skelDir, "user.skel"), `
domain demo.user

actor ClientActor { via client {} }

service UserService {
    for ClientActor

    method getUser {
        output User
    }
}

data User {
    id: string
}

config SiteConfig eternal {
    title: string
}
`)

	result := Parse(Option{SkelIn: skelDir})

	if result.Domain.Name() != "demo.user" {
		t.Fatalf("unexpected domain name: %s", result.Domain.Name())
	}
	if result.Domain.Description() != "User domain" {
		t.Fatalf("unexpected domain description: %s", result.Domain.Description())
	}
	if len(result.Domain.Data()) != 1 || result.Domain.Data()[0].Name != "User" {
		t.Fatalf("unexpected data: %#v", result.Domain.Data())
	}
	if len(result.Domain.Configs()) != 1 || result.Domain.Configs()[0].Name != "SiteConfig" {
		t.Fatalf("unexpected configs: %#v", result.Domain.Configs())
	}
	if len(result.Domain.Services()) != 1 || result.Domain.Services()[0].Name != "UserService" {
		t.Fatalf("unexpected services: %#v", result.Domain.Services())
	}
}

func TestParseImportedDomainDoesNotRequireTransitiveImports(t *testing.T) {
	root := t.TempDir()
	appDir := filepath.Join(root, "app")
	userDir := filepath.Join(root, "user")
	bookerDir := filepath.Join(root, "booker")
	for _, dir := range []string{appDir, userDir, bookerDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}

	writeParseFile(t, filepath.Join(appDir, "domain.skel"), "domain app\n")
	writeParseFile(t, filepath.Join(appDir, "actor.skel"), `
domain app

pub actor UserActor {
    via client {}
}
`)
	writeParseFile(t, filepath.Join(userDir, "domain.skel"), "domain user\n")
	writeParseFile(t, filepath.Join(userDir, "service.skel"), `
domain user

import app

pub service UserService {
    for app.UserActor

    method getUser {
        output UserSummary
    }
}

pub data UserSummary {
    id: string
}
`)
	writeParseFile(t, filepath.Join(bookerDir, "domain.skel"), "domain booker\n")
	writeParseFile(t, filepath.Join(bookerDir, "types.skel"), `
domain booker

import user

pub data Loan {
    borrower: user.UserSummary
}
`)

	result := Parse(Option{
		SkelIn:      bookerDir,
		SkelImports: map[string]string{"user": userDir},
	})

	if len(result.Domain.Data()) != 1 || result.Domain.Data()[0].Name != "Loan" {
		t.Fatalf("unexpected data: %#v", result.Domain.Data())
	}
	if len(result.Domain.Imports()) != 1 || result.Domain.Imports()[0].Name != "user" {
		t.Fatalf("unexpected imports: %#v", result.Domain.Imports())
	}
}

func TestParseImportDoesNotRequireImportedDomains(t *testing.T) {
	skelDir := t.TempDir()
	writeParseFile(t, filepath.Join(skelDir, "domain.skel"), "domain demo.booker\n")
	writeParseFile(t, filepath.Join(skelDir, "types.skel"), `
domain demo.booker

import demo.user as user

data Booking {
    owner: user.User
}
`)

	result := ParseImport(skelDir)

	if result.Domain.Name() != "demo.booker" {
		t.Fatalf("unexpected domain name: %s", result.Domain.Name())
	}
}

func TestParsePanicsWhenDomainFileMissing(t *testing.T) {
	skelDir := t.TempDir()
	writeParseFile(t, filepath.Join(skelDir, "user.skel"), "data User { id: string }\n")

	expectParsePanicContains(t, "domain.skel not found", func() {
		Parse(Option{SkelIn: skelDir})
	})
}

func TestParseSingleSkelFile(t *testing.T) {
	skelFile := filepath.Join(t.TempDir(), "user.skel")
	writeParseFile(t, skelFile, `@desc("User domain")
domain demo.user

data User {
    id: string
}
`)

	result := Parse(Option{SkelIn: skelFile})

	if result.Domain.Name() != "demo.user" || len(result.Domain.Data()) != 1 {
		t.Fatalf("unexpected domain: %#v", result.Domain)
	}
}

func TestParsePanicsForInvalidSkel(t *testing.T) {
	skelFile := filepath.Join(t.TempDir(), "user.skel")
	writeParseFile(t, skelFile, `domain demo.user

actor ClientActor { via client {} }

service UserService {
    for ClientActor

    method getUser {
        output MissingUser
    }
}
`)

	expectParsePanicContains(t, "definition of MissingUser not found", func() {
		Parse(Option{SkelIn: skelFile})
	})
}

func expectParsePanicContains(t *testing.T, expected string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic containing %q", expected)
		}
		if !strings.Contains(fmt.Sprint(recovered), expected) {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()
	fn()
}

func writeParseFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
