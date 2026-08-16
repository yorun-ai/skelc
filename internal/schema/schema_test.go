package schema

import (
	"bytes"
	"strings"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestEncodeDecodeRoundTripOmitsSourcePositions(t *testing.T) {
	document := newTestDocument(
		&Declaration{
			Pub: true, Name: "User", Kind: "data", SkelName: "demo.user.User",
			Pos: model.Position{File: "/workspace/user.skel", Line: 2, Column: 10},
			Data: &DataSchema{Members: []*Member{{
				Name: "id", Type: scalarType("string"),
				Pos: model.Position{File: "/workspace/user.skel", Line: 3, Column: 5},
			}}},
		},
	)
	var output bytes.Buffer
	if err := Encode(&output, document); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "/workspace") {
		t.Fatalf("encoded schema contains source position:\n%s", output.String())
	}
	decoded, err := Decode(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Domain != document.Domain || len(decoded.Declarations) != 1 {
		t.Fatalf("unexpected decoded schema: %+v", decoded)
	}
	if decoded.Declarations[0].Pos != (model.Position{}) {
		t.Fatalf("decoded source position should be empty: %+v", decoded.Declarations[0].Pos)
	}
}

func TestDecodeRejectsUnknownAndIncompleteFields(t *testing.T) {
	for _, input := range []string{
		`{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","unknown":true,"declarations":[]}`,
		`{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo"}`,
		`{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","declarations":[{"pub":true,"name":"User","type":"data","skelName":"demo.User","data":{"members":[{"name":"id","type":null}]}}]}`,
	} {
		if _, err := Decode(strings.NewReader(input)); err == nil {
			t.Fatalf("expected invalid schema to fail: %s", input)
		}
	}
}

func TestCompareClassifiesAndOrdersChanges(t *testing.T) {
	baseline := newTestDocument(
		dataDeclaration("User", "id"),
		enumDeclaration("UserStatus", "ACTIVE"),
		serviceDeclaration("UserService", "getUser"),
	)
	candidate := newTestDocument(
		dataDeclaration("User", "id", "name"),
		enumDeclaration("UserStatus", "ACTIVE", "DISABLED"),
		serviceDeclaration("UserService", "listUsers"),
	)
	report, err := Diff(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if report.Compatible {
		t.Fatal("expected report to be incompatible")
	}
	if report.Summary != (Summary{Breaking: 2, Dangerous: 1, Compatible: 1}) {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	codes := make([]string, 0, len(report.Changes))
	for _, change := range report.Changes {
		codes = append(codes, change.Code)
	}
	expected := []string{
		"data.member.added",
		"service.method.removed",
		"enum.item.added",
		"service.method.added",
	}
	if strings.Join(codes, ",") != strings.Join(expected, ",") {
		t.Fatalf("unexpected changes: %v", codes)
	}
	if report.Changes[0].Change != ChangeAdded || report.Changes[1].Change != ChangeRemoved ||
		report.Changes[2].Change != ChangeAdded || report.Changes[3].Change != ChangeAdded {
		t.Fatalf("unexpected change kinds: %+v", report.Changes)
	}
}

func TestCompareTreatsDocumentationAsCompatible(t *testing.T) {
	baseline := newTestDocument(dataDeclaration("User", "id"))
	candidate := newTestDocument(dataDeclaration("User", "id"))
	candidate.Declarations[0].Description = "User data"
	report, err := Diff(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Compatible || report.Summary.Compatible != 1 || report.Changes[0].Change != ChangeModified ||
		report.Changes[0].Code != "declaration.description.changed" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestCompareRecognizesDeclarationTypeChangeWithoutNamespaceCollision(t *testing.T) {
	baseline := newTestDocument(dataDeclaration("State", "value"))
	candidate := newTestDocument(enumDeclaration("State", "ACTIVE"))
	report, err := Diff(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Breaking != 1 || len(report.Changes) != 1 || report.Changes[0].Code != "declaration.type.changed" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestCompareTreatsAuthenticationAndPermissionSemanticsAsDangerous(t *testing.T) {
	readRequirement := func() *Requirement {
		return &Requirement{Mode: "code", Code: "identity.User:read"}
	}
	writeRequirement := func() *Requirement {
		return &Requirement{Mode: "code", Code: "identity.User:write"}
	}
	tests := []struct {
		name      string
		baseline  *Document
		candidate *Document
		code      string
	}{
		{"actor authentication added", actorDocument(false, false), actorDocument(true, false), "actor.auth.added"},
		{"actor authentication removed", actorDocument(true, false), actorDocument(false, false), "actor.auth.removed"},
		{"actor permission added", actorDocument(false, false), actorDocument(false, true), "actor.permission.added"},
		{"actor permission removed", actorDocument(false, true), actorDocument(false, false), "actor.permission.removed"},
		{"resource permission code changed", resourceDocument("identity.User:read"), resourceDocument("identity.User:write"), "resource.action.code.changed"},
		{"service authentication tightened", servicePolicyDocument("unset", nil), servicePolicyDocument("auth", nil), "service.auth.tightened"},
		{"service authentication relaxed", servicePolicyDocument("auth", nil), servicePolicyDocument("unset", nil), "service.auth.relaxed"},
		{"service permission added", servicePolicyDocument("unset", nil), servicePolicyDocument("unset", readRequirement()), "service.require.added"},
		{"service permission changed", servicePolicyDocument("unset", readRequirement()), servicePolicyDocument("unset", writeRequirement()), "service.require.changed"},
		{"service permission removed", servicePolicyDocument("unset", readRequirement()), servicePolicyDocument("unset", nil), "service.require.removed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report, err := Diff(test.baseline, test.candidate)
			if err != nil {
				t.Fatal(err)
			}
			if !report.Compatible || report.Summary != (Summary{Dangerous: 1}) || len(report.Changes) != 1 ||
				report.Changes[0].Impact != ImpactDangerous || report.Changes[0].Code != test.code {
				t.Fatalf("unexpected report: %+v", report)
			}
		})
	}
}

func newTestDocument(declarations ...*Declaration) *Document {
	return &Document{
		Format: Format, FormatVersion: FormatVersion, Domain: "demo.user",
		Declarations: declarations,
	}
}

func dataDeclaration(name string, members ...string) *Declaration {
	values := make([]*Member, 0, len(members))
	for _, member := range members {
		values = append(values, &Member{Name: member, Type: scalarType("string")})
	}
	return &Declaration{
		Pub: true, Name: name, Kind: "data", SkelName: "demo.user." + name,
		Data: &DataSchema{Members: values},
	}
}

func enumDeclaration(name string, items ...string) *Declaration {
	values := make([]*EnumItem, 0, len(items))
	for _, item := range items {
		values = append(values, &EnumItem{Name: item})
	}
	return &Declaration{
		Pub: true, Name: name, Kind: "enum", SkelName: "demo.user." + name,
		Enum: &EnumSchema{Items: values},
	}
}

func serviceDeclaration(name string, methods ...string) *Declaration {
	values := make([]*Method, 0, len(methods))
	for _, method := range methods {
		values = append(values, &Method{Name: method, SkelName: method, Auth: "unset", Arguments: []*Argument{}})
	}
	return &Declaration{
		Pub: true, Name: name, Kind: "service", SkelName: "demo.user." + name,
		Service: &ServiceSchema{Auth: "unset", Audiences: []*Audience{}, Methods: values},
	}
}

func actorDocument(authEnabled, permEnabled bool) *Document {
	actor := &ActorSchema{Vias: []*ActorVia{}, AuthEnabled: authEnabled, PermEnabled: permEnabled}
	if authEnabled {
		actor.AuthCredential = &DataSchema{Members: []*Member{}}
		actor.AuthInfo = &DataSchema{Members: []*Member{}}
	}
	return newTestDocument(&Declaration{
		Pub: true, Name: "UserActor", Kind: "actor", SkelName: "demo.user.UserActor", Actor: actor,
	})
}

func resourceDocument(permissionCode string) *Document {
	return newTestDocument(&Declaration{
		Pub: true, Name: "User", Kind: "resource", SkelName: "demo.user.User",
		Resource: &ResourceSchema{Actions: []*ResourceAction{{
			Name: "read", PermissionCode: permissionCode, Checks: []*ResourceCheck{},
		}}},
	})
}

func servicePolicyDocument(auth string, require *Requirement) *Document {
	declaration := serviceDeclaration("UserService")
	declaration.Service.Auth = auth
	declaration.Service.Require = require
	return newTestDocument(declaration)
}

func scalarType(name string) *Type {
	return &Type{Kind: "scalar", Name: name}
}
