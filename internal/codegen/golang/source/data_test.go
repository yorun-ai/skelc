package source

import (
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestCastData(t *testing.T) {
	data := castData(&model.Data{
		Name:        "Page",
		Description: "Paginated result",
		Sensitive:   true,
		TypeParameters: []*model.TypeParameter{
			{Name: "TItem"},
		},
		Members: []*model.DataMember{
			{
				Name:        "generatedAt",
				Description: "Generated at",
				Type: &model.Type{
					Kind:   model.TypeKindScalar,
					Scalar: model.ScalarTimestamp,
				},
			},
			{
				Name:        "avatarUrl",
				Description: "Avatar URL",
				Example:     `"https://xxx.com/a.png"`,
				Type: &model.Type{
					Kind:     model.TypeKindScalar,
					Scalar:   model.ScalarString,
					Nullable: true,
				},
			},
		},
	})

	if data.FullName != "Page[TItem any]" {
		t.Fatalf("unexpected full name: %s", data.FullName)
	}
	if !data.Sensitive {
		t.Fatal("expected sensitive data marker")
	}
	if len(data.CommentLines) == 0 || data.CommentLines[0] != "Page Paginated result" {
		t.Fatalf("unexpected data comment lines: %+v", data.CommentLines)
	}
	if len(data.Members) != 2 {
		t.Fatalf("unexpected member count: %d", len(data.Members))
	}
	if data.Members[0].Type.Plain != "skel.Timestamp" {
		t.Fatalf("unexpected first member type: %s", data.Members[0].Type.Plain)
	}
	if len(data.Members[1].CommentLines) == 0 || data.Members[1].CommentLines[0] != `AvatarUrl Avatar URL (e.g. "https://xxx.com/a.png")` {
		t.Fatalf("unexpected second member comment lines: %+v", data.Members[1].CommentLines)
	}
}

func TestCastDataBuildsCloneMethod(t *testing.T) {
	child := &model.Data{
		Name:    "Child",
		Members: []*model.DataMember{{Name: "name", Type: stringTypeForTest()}},
	}
	data := castCloneableData(&model.Data{
		Name: "Payload",
		Members: []*model.DataMember{
			{Name: "content", Type: scalarTypeForTest(model.ScalarBinary)},
			{Name: "children", Type: listTypeForTest(dataTypeForTest(child))},
			{Name: "labels", Type: mapTypeForTest(stringTypeForTest(), stringTypeForTest())},
		},
	})

	if !data.Clone || data.CloneMethodName != "Clone" || data.CloneParameters != "" {
		t.Fatalf("unexpected clone metadata: %+v", data)
	}
	lines := strings.Join(data.CloneLines, "\n")
	for _, fragment := range []string{
		"cloned.Content = append(v.Content[:0:0], v.Content...)",
		"cloned.Children[index0] = v.Children[index0].Clone()",
		"cloned.Labels = maps.Clone(v.Labels)",
	} {
		if !strings.Contains(lines, fragment) {
			t.Fatalf("clone lines missing %q:\n%s", fragment, lines)
		}
	}
	if got := importPaths(data.CloneImports); len(got) != 1 || got[0] != "maps" {
		t.Fatalf("unexpected clone imports: %v", got)
	}
}

func TestCastGenericDataBuildsCloneByMethod(t *testing.T) {
	tItem := typeParamForTest("TItem")
	data := castCloneableData(&model.Data{
		Name:           "Page",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: listTypeForTest(typeParamTypeForTest(tItem))},
		},
	})

	if !data.Clone || data.CloneMethodName != "CloneBy" {
		t.Fatalf("unexpected clone metadata: %+v", data)
	}
	if data.CloneParameters != "cloneTItem func(TItem) TItem" {
		t.Fatalf("unexpected clone parameters: %q", data.CloneParameters)
	}
	if lines := strings.Join(data.CloneLines, "\n"); !strings.Contains(lines, "cloned.Items[index0] = cloneTItem(v.Items[index0])") {
		t.Fatalf("generic clone lines did not use callback:\n%s", lines)
	}
}

func TestCastDataCallsNestedGenericCloneBy(t *testing.T) {
	tItem := typeParamForTest("TItem")
	page := &model.Data{
		Name:           "Page",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: listTypeForTest(typeParamTypeForTest(tItem))},
		},
	}
	user := &model.Data{Name: "User", Members: []*model.DataMember{{Name: "name", Type: stringTypeForTest()}}}
	data := castCloneableData(&model.Data{
		Name: "Users",
		Members: []*model.DataMember{
			{Name: "page", Type: dataTypeForTest(page, dataTypeForTest(user))},
		},
	})

	lines := strings.Join(data.CloneLines, "\n")
	if !strings.Contains(lines, "v.Page.CloneBy(func(value User) User {") ||
		!strings.Contains(lines, "return value.Clone()") {
		t.Fatalf("nested generic clone did not build concrete callback:\n%s", lines)
	}
}

func TestCastDataDoesNotGenerateConflictingCloneMethod(t *testing.T) {
	data := castCloneableData(&model.Data{
		Name:    "Value",
		Members: []*model.DataMember{{Name: "clone", Type: stringTypeForTest()}},
	})
	if data.Clone {
		t.Fatalf("data field Clone must prevent Clone method generation: %+v", data)
	}
}

func TestCastDataDoesNotGenerateCloneForImportedMember(t *testing.T) {
	external := &model.Data{Name: "User", Domain: "identity.user"}
	externalType := dataTypeForTest(external)
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
	data := castCloneableData(&model.Data{
		Name:    "Order",
		Members: []*model.DataMember{{Name: "user", Type: externalType}},
	})
	if data.Clone {
		t.Fatalf("data with imported member must retain serialization fallback: %+v", data)
	}
}

func TestCastDataDoesNotGenerateCloneForRecursiveData(t *testing.T) {
	node := &model.Data{Name: "Node"}
	node.Members = []*model.DataMember{{Name: "children", Type: listTypeForTest(dataTypeForTest(node))}}
	data := castCloneableData(node)
	if data.Clone {
		t.Fatalf("recursive data must retain serialization fallback: %+v", data)
	}
}

func TestBuildDataImports(t *testing.T) {
	imports := buildDataImports([]*Data{
		{
			Sensitive: true,
			Members: []*DataMember{
				{Type: &Type{Imports: []*Import{{Path: skelImport}}}},
				{Type: &Type{Imports: []*Import{{Path: skelImport}}}},
			},
		},
	})

	if got, want := importPaths(imports), []string{skelImport}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected import paths: got=%v want=%v", got, want)
	}
}

func TestSensitiveMarkerMethodNeedsNoImport(t *testing.T) {
	if imports := buildDataImports([]*Data{{Sensitive: true}}); len(imports) != 0 {
		t.Fatalf("unexpected imports for sensitive marker method: %v", importPaths(imports))
	}
}

func TestCastDataMapsDurationToSkelDuration(t *testing.T) {
	data := castData(&model.Data{
		Name: "TimeoutConfig",
		Members: []*model.DataMember{
			{
				Name: "timeout",
				Type: &model.Type{
					Kind:   model.TypeKindScalar,
					Scalar: model.ScalarDuration,
				},
			},
		},
	})
	if data.Members[0].Type.Plain != "skel.Duration" {
		t.Fatalf("unexpected duration member type: %s", data.Members[0].Type.Plain)
	}
}

func TestCastDataMapsLocalDateToSkelLocalDate(t *testing.T) {
	data := castData(&model.Data{
		Name: "Profile",
		Members: []*model.DataMember{
			{
				Name: "birthday",
				Type: &model.Type{
					Kind:   model.TypeKindScalar,
					Scalar: model.ScalarLocalDate,
				},
			},
		},
	})
	if data.Members[0].Type.Plain != "skel.LocalDate" {
		t.Fatalf("unexpected date member type: %s", data.Members[0].Type.Plain)
	}
}
