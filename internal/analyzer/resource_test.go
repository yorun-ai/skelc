package analyzer

import (
	"go.yorun.ai/skelc/internal/model"
	"testing"
)

func TestResourceDefaultsToNonPub(t *testing.T) {
	content := parseResourceTestContent(t, `
domain demo

resource User {
    action read
}
`)
	domain := mustAnalyze(t, content).Model()

	if len(domain.Resources()) != 1 {
		t.Fatalf("unexpected resource count: %d", len(domain.Resources()))
	}
	if domain.Resources()[0].Pub {
		t.Fatalf("resource Pub = true, want false")
	}
}

func TestPubResourceSetsPub(t *testing.T) {
	content := parseResourceTestContent(t, `
domain demo

pub resource User {
    action read
}
`)
	domain := mustAnalyze(t, content).Model()

	if len(domain.Resources()) != 1 {
		t.Fatalf("unexpected resource count: %d", len(domain.Resources()))
	}
	if !domain.Resources()[0].Pub {
		t.Fatalf("resource Pub = false, want true")
	}
}

func TestRequireRejectsImportedNonPubResource(t *testing.T) {
	imported := mustAnalyze(t, parseResourceTestContent(t, `
domain app

resource User {
    action read
}
`))
	content := parseResourceTestContent(t, `
domain user

import app

actor UserActor {
    via client {}
}

service UserService {
    for UserActor

    method getUser {
        require app.User:read
    }
}
`)

	_, diagnostics := Analyze(content, []*Analysis{imported})
	assertDiagnosticsContain(t, diagnostics, "references non-pub resource")
}

// TODO: Remove this regression test after the PermissionCode migration period ends.
func TestPermissionCodeIsNotABuiltinType(t *testing.T) {
	for _, declaration := range []string{
		`data Payload { code: PermissionCode }`,
		`resource User {
    check byId { input { permission: PermissionCode
        id: string
    } }
    action read
}`,
	} {
		content := parseResourceTestContent(t, "domain demo\n"+declaration)
		_, diagnostics := Analyze(content, nil)
		assertDiagnosticsContain(t, diagnostics, "PermissionCode")
	}
}

func TestResourceCheckRejectsExplicitStringCode(t *testing.T) {
	content := parseResourceTestContent(t, `domain demo
resource User {
    check byId { input { code: string } }
    action read
}`)
	_, diagnostics := Analyze(content, nil)
	assertDiagnosticsContain(t, diagnostics, `argument name "code" is reserved`)
}

// TODO: Remove this regression test after the PermissionCode migration period ends.
func TestResourceCheckAllowsUserDefinedPermissionCodeData(t *testing.T) {
	content := parseResourceTestContent(t, `domain demo
data PermissionCode { value: string }
resource User {
    check byPermission { input { permission: PermissionCode } }
    action read
}
actor UserActor { via client {} }
service UserService {
    for UserActor
    method read {
        require User:read:byPermission(permission)
        input { permission: PermissionCode }
    }
}`)
	domain := mustAnalyze(t, content).Model()
	check := domain.Resources()[0].Checks[0]
	arguments := resourceCheckArguments(check)
	if len(arguments) != 1 || arguments[0].Name != "permission" || arguments[0].Source != model.ArgumentSourceDeclared {
		t.Fatalf("user-defined data argument was treated as injected: %+v", arguments)
	}
	if arguments[0].Type.Kind != model.TypeKindData || arguments[0].Type.Data.Name != "PermissionCode" {
		t.Fatalf("expected an ordinary data reference, got %+v", arguments[0].Type)
	}
}
