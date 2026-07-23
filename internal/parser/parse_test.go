package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/model"
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

	result, err := Parse(Option{SkelIn: skelDir})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

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

	result, err := Parse(Option{
		SkelIn:      bookerDir,
		SkelImports: map[string]string{"user": userDir},
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Domain.Data()) != 1 || result.Domain.Data()[0].Name != "Loan" {
		t.Fatalf("unexpected data: %#v", result.Domain.Data())
	}
	if len(result.Domain.Imports()) != 1 || result.Domain.Imports()[0].Name != "user" {
		t.Fatalf("unexpected imports: %#v", result.Domain.Imports())
	}
}

func TestParseNormalizesImportedDomainLocalTypeReferences(t *testing.T) {
	root := t.TempDir()
	baseDir := filepath.Join(root, "base")
	appDir := filepath.Join(root, "app")
	for _, dir := range []string{baseDir, appDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create %s: %v", dir, err)
		}
	}

	writeParseFile(t, filepath.Join(baseDir, "domain.skel"), "domain base\n")
	writeParseFile(t, filepath.Join(baseDir, "types.skel"), `
domain base

pub enum ItemType {
    STANDARD
}

pub data Detail {
    name: string
}

pub data Box<TItem> {
    value: TItem
}

pub data Item {
    type: ItemType
    detail: Detail
    boxedType: Box<ItemType>
}
`)
	writeParseFile(t, filepath.Join(appDir, "domain.skel"), "domain app\n")
	writeParseFile(t, filepath.Join(appDir, "types.skel"), `
domain app

import base

data AppItem {
    item: base.Item
}
`)

	result, err := Parse(Option{
		SkelIn:      appDir,
		SkelImports: map[string]string{"base": baseDir},
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	baseDomain := result.Domain.Imports()[0].Domain
	var box, item *model.Data
	for _, dataType := range baseDomain.Data() {
		switch dataType.Name {
		case "Box":
			box = dataType
		case "Item":
			item = dataType
		}
	}
	if box == nil || item == nil {
		t.Fatalf("expected imported Box and Item data: %+v", baseDomain.Data())
	}
	if got := box.Members[0].Type.Kind; got != model.TypeKindTypeParameter {
		t.Fatalf("generic member kind = %d, want %d", got, model.TypeKindTypeParameter)
	}
	wantKinds := []model.TypeKind{model.TypeKindEnum, model.TypeKindData, model.TypeKindData}
	for index, member := range item.Members {
		if member.Type.Kind != wantKinds[index] {
			t.Fatalf("member %s kind = %d, want %d", member.Name, member.Type.Kind, wantKinds[index])
		}
	}
	if got := item.Members[2].Type.TypeArguments[0].Kind; got != model.TypeKindEnum {
		t.Fatalf("generic type argument kind = %d, want %d", got, model.TypeKindEnum)
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

	result, err := ParseImport(skelDir)
	if err != nil {
		t.Fatalf("ParseImport() error = %v", err)
	}

	if result.Domain.Name() != "demo.booker" {
		t.Fatalf("unexpected domain name: %s", result.Domain.Name())
	}
}

func TestParseReturnsErrorWhenDomainFileMissing(t *testing.T) {
	skelDir := t.TempDir()
	writeParseFile(t, filepath.Join(skelDir, "user.skel"), "data User { id: string }\n")

	_, err := Parse(Option{SkelIn: skelDir})
	expectErrorContains(t, err, "domain.skel not found")
}

func TestParseSingleSkelFile(t *testing.T) {
	skelFile := filepath.Join(t.TempDir(), "user.skel")
	writeParseFile(t, skelFile, `@desc("User domain")
domain demo.user

data User {
    id: string
}
`)

	result, err := Parse(Option{SkelIn: skelFile})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

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

	_, err := Parse(Option{SkelIn: skelFile})
	if err == nil || !strings.Contains(err.Error(), "definition of MissingUser not found") {
		t.Fatalf("expected semantic error, got %v", err)
	}
}

func writeParseFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
