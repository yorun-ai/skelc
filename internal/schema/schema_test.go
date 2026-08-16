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
		`{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","scope":"public","unknown":true,"declarations":[]}`,
		`{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","scope":"public"}`,
		`{"format":"yorun.skel.schema","formatVersion":1,"domain":"demo","scope":"public","declarations":[{"pub":true,"name":"User","type":"data","skelName":"demo.User","data":{"members":[{"name":"id","type":null}]}}]}`,
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
	report, err := Compare(baseline, candidate)
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
}

func TestCompareTreatsDocumentationAsCompatible(t *testing.T) {
	baseline := newTestDocument(dataDeclaration("User", "id"))
	candidate := newTestDocument(dataDeclaration("User", "id"))
	candidate.Declarations[0].Description = "User data"
	report, err := Compare(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Compatible || report.Summary.Compatible != 1 || report.Changes[0].Code != "declaration.description.changed" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestCompareRecognizesDeclarationTypeChangeWithoutNamespaceCollision(t *testing.T) {
	baseline := newTestDocument(dataDeclaration("State", "value"))
	candidate := newTestDocument(enumDeclaration("State", "ACTIVE"))
	report, err := Compare(baseline, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Breaking != 1 || len(report.Changes) != 1 || report.Changes[0].Code != "declaration.type.changed" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func newTestDocument(declarations ...*Declaration) *Document {
	return &Document{
		Format: Format, FormatVersion: FormatVersion, Domain: "demo.user", Scope: ScopePublic,
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

func scalarType(name string) *Type {
	return &Type{Kind: "scalar", Name: name}
}
