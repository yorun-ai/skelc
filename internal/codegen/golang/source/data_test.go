package source

import (
	"reflect"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/model"
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

	if !data.Clone || data.CloneMethodName != "Clone" || len(data.CloneParameters) != 0 {
		t.Fatalf("unexpected clone metadata: %+v", data)
	}
	lines := renderGoIRForTest(t, "goBlock", data.CloneBlock)
	for _, fragment := range []string{
		"if v.Content == nil {",
		"cloned.Content = make(skel.Binary, len(v.Content))",
		"copy(cloned.Content, v.Content)",
		"cloned.Children = make([]Child, len(v.Children))",
		"cloned.Children[index0] = v.Children[index0].Clone()",
		"cloned.Labels = maps.Clone(v.Labels)",
	} {
		if !strings.Contains(lines, fragment) {
			t.Fatalf("clone lines missing %q:\n%s", fragment, lines)
		}
	}
	if strings.Contains(lines, "[:0:0]") {
		t.Fatalf("clone lines retain source slice backing arrays:\n%s", lines)
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
	if len(data.CloneParameters) != 1 ||
		data.CloneParameters[0].Name != "cloneTItem" ||
		data.CloneParameters[0].Type != "func(TItem) TItem" {
		t.Fatalf("unexpected clone parameters: %+v", data.CloneParameters)
	}
	if lines := renderGoIRForTest(t, "goBlock", data.CloneBlock); !strings.Contains(lines, "cloned.Items[index0] = cloneTItem(v.Items[index0])") {
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

	lines := renderGoIRForTest(t, "goBlock", data.CloneBlock)
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

func TestCastDataBuildsCompatibleCloneForImportedMember(t *testing.T) {
	external := &model.Data{Name: "User", Domain: "identity.user"}
	externalType := dataTypeForTest(external)
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
	externalType.ExternalAlias = "userpub"
	data := castCloneableData(&model.Data{
		Name:    "Order",
		Members: []*model.DataMember{{Name: "user", Type: externalType}},
	})
	if !data.Clone {
		t.Fatalf("data with imported member must expose Clone: %+v", data)
	}
	lines := renderGoIRForTest(t, "goBlock", data.CloneBlock)
	for _, fragment := range []string{
		"any(value).(interface { Clone() userpub.User })",
		"return cloner0.Clone()",
		"return vcode.MustUnmarshalJson[userpub.User](vcode.MustMarshalJson(value))",
	} {
		if !strings.Contains(lines, fragment) {
			t.Fatalf("compatible imported clone missing %q:\n%s", fragment, lines)
		}
	}
	if got, want := importPaths(data.CloneImports), []string{"example.com/identity", "go.yorun.ai/vine/util/vcode"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected imported clone imports: got=%v want=%v", got, want)
	}
}

func TestCastDataBuildsCloneForRecursiveData(t *testing.T) {
	node := &model.Data{Name: "Node"}
	node.Members = []*model.DataMember{{Name: "children", Type: listTypeForTest(dataTypeForTest(node))}}
	data := castCloneableData(node)
	if !data.Clone {
		t.Fatalf("recursive data must expose Clone: %+v", data)
	}
	if lines := renderGoIRForTest(t, "goBlock", data.CloneBlock); !strings.Contains(lines, "cloned.Children[index0] = v.Children[index0].Clone()") {
		t.Fatalf("recursive clone missing nested Clone call:\n%s", lines)
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
