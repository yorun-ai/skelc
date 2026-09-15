package source

import (
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/codegentest"
	"go.yorun.ai/skelc/internal/model"
)

func TestCastServiceMethodBuildsTypedDeepCloneHooks(t *testing.T) {
	payloadType := &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarBinary, Nullable: true}
	child := &model.Data{
		Name:    "Child",
		Members: []*model.DataMember{{Name: "name", Type: codegentest.StringType()}},
	}
	node := &model.Data{Name: "Node", Members: []*model.DataMember{
		{Name: "payload", Type: payloadType},
		{Name: "children", Type: codegentest.ListType(codegentest.DataType(child))},
		{Name: "labels", Type: codegentest.MapType(codegentest.StringType(), codegentest.StringType())},
	}}
	parsed := &model.Method{
		Name: "clone",
		Arguments: []*model.Argument{
			{Name: "node", Type: codegentest.NullableType(codegentest.DataType(node))},
		},
		ArgumentsData: &model.Data{
			Name: "CloneServiceCloneArguments",
			Members: []*model.DataMember{
				{Name: "node", Type: codegentest.NullableType(codegentest.DataType(node))},
			},
		},
		ResultType: codegentest.DataType(node),
	}

	method := castServiceMethod(&model.Service{Name: "CloneService"}, parsed)
	cloneArguments := renderGoIRForTest(t, "goFunction", method.CloneArguments)
	cloneResult := renderGoIRForTest(t, "goFunction", method.CloneResult)

	for _, fragment := range []string{
		"source := value.(*_CloneServiceCloneArguments)",
		"clonedValue0 = (*source.Node).Clone()",
		"cloned.Node = &clonedValue",
	} {
		if !strings.Contains(cloneArguments, fragment) {
			t.Fatalf("CloneArguments missing %q:\n%s", fragment, cloneArguments)
		}
	}
	if !strings.Contains(cloneResult, "source := value.(Node)") ||
		!strings.Contains(cloneResult, "cloned = source.Clone()") {
		t.Fatalf("unexpected CloneResult:\n%s", cloneResult)
	}
	if got := importPaths(method.CloneImports); len(got) != 0 {
		t.Fatalf("unexpected clone imports: %v", got)
	}
}

func TestCastServiceMethodCallsCloneForImportedData(t *testing.T) {
	external := &model.Data{Name: "User", Domain: "identity.user"}
	externalType := codegentest.DataType(external)
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
	externalType.ExternalAlias = "userpub"
	parsed := &model.Method{
		Name: "get",
		Arguments: []*model.Argument{
			{Name: "user", Type: externalType},
		},
		ArgumentsData: &model.Data{
			Name:    "UserServiceGetArguments",
			Members: []*model.DataMember{{Name: "user", Type: externalType}},
		},
		ResultType: externalType,
	}

	method := castServiceMethod(&model.Service{Name: "UserService"}, parsed)

	if method.CloneArguments == nil || method.CloneResult == nil {
		t.Fatalf("imported data must use typed clone hooks: %+v", method)
	}
	arguments := renderGoIRForTest(t, "goFunction", method.CloneArguments)
	if !strings.Contains(arguments, "cloned.User = source.User.Clone()") {
		t.Fatalf("argument clone missing direct Clone call:\n%s", arguments)
	}
	result := renderGoIRForTest(t, "goFunction", method.CloneResult)
	if !strings.Contains(result, "cloned = source.Clone()") {
		t.Fatalf("result clone missing direct Clone call:\n%s", result)
	}
}

func TestCastServiceMethodBuildsCloneForRecursiveData(t *testing.T) {
	node := &model.Data{Name: "Node"}
	node.Members = []*model.DataMember{{Name: "children", Type: codegentest.ListType(codegentest.DataType(node))}}
	parsed := &model.Method{
		Name:       "get",
		ResultType: codegentest.DataType(node),
	}

	method := castServiceMethod(&model.Service{Name: "NodeService"}, parsed)

	if method.CloneResult == nil {
		t.Fatalf("recursive data must use its generated Clone method: %+v", method)
	}
	if rendered := renderGoIRForTest(t, "goFunction", method.CloneResult); !strings.Contains(rendered, "cloned = source.Clone()") {
		t.Fatalf("recursive result clone missing Clone call:\n%s", rendered)
	}
}

func TestCastServiceMethodCallsCloneByForImportedGenericData(t *testing.T) {
	tItem := &model.TypeParameter{Name: "TItem"}
	externalPage := &model.Data{
		Name:           "Page",
		Domain:         "identity.user",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: codegentest.ListType(codegentest.TypeParamType(tItem))},
		},
	}
	externalType := codegentest.DataType(externalPage, codegentest.StringType())
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
	externalType.ExternalAlias = "userpub"

	method := castServiceMethod(&model.Service{Name: "PageService"}, &model.Method{
		Name:       "list",
		ResultType: externalType,
	})

	if method.CloneResult == nil {
		t.Fatalf("imported generic data must use a typed clone hook: %+v", method)
	}
	rendered := renderGoIRForTest(t, "goFunction", method.CloneResult)
	if !strings.Contains(rendered, "cloned = source.CloneBy(func(value string) string { return value })") {
		t.Fatalf("imported generic clone missing direct CloneBy call:\n%s", rendered)
	}
}

func TestCastServiceMethodCallsCloneByForImportedGenericDataWithUnresolvedTransitiveType(t *testing.T) {
	tItem := &model.TypeParameter{Name: "TItem"}
	transitive := &model.Data{Name: "Meta", Domain: "shared.meta"}
	transitiveType := codegentest.DataType(transitive)
	transitiveType.ExternalDomain = "shared.meta"
	externalPage := &model.Data{
		Name:           "Page",
		Domain:         "identity.user",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: codegentest.ListType(codegentest.TypeParamType(tItem))},
			{Name: "metadata", Type: codegentest.ListType(transitiveType)},
		},
	}
	externalType := codegentest.DataType(externalPage, codegentest.StringType())
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
	externalType.ExternalAlias = "userpub"

	method := castServiceMethod(&model.Service{Name: "PageService"}, &model.Method{
		Name:       "list",
		ResultType: externalType,
	})
	rendered := renderGoIRForTest(t, "goFunction", method.CloneResult)
	if !strings.Contains(rendered, "cloned = source.CloneBy(func(value string) string { return value })") {
		t.Fatalf("imported generic clone missing direct CloneBy call:\n%s", rendered)
	}
}

func TestCastServiceMethodBuildsGenericDataClone(t *testing.T) {
	tItem := &model.TypeParameter{Name: "TItem"}
	page := &model.Data{
		Name:           "Page",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: codegentest.ListType(&model.Type{Kind: model.TypeKindTypeParameter, TypeParameter: tItem})},
		},
	}
	resultType := codegentest.DataType(page, codegentest.StringType())
	method := castServiceMethod(&model.Service{Name: "PageService"}, &model.Method{Name: "list", ResultType: resultType})

	if method.CloneResult == nil {
		t.Fatalf("expected generic result clone: %+v", method)
	}
	cloneResult := renderGoIRForTest(t, "goFunction", method.CloneResult)
	for _, fragment := range []string{
		"source.CloneBy(func(value string) string { return value })",
	} {
		if !strings.Contains(cloneResult, fragment) {
			t.Fatalf("generic clone hook missing %q:\n%s", fragment, cloneResult)
		}
	}
}

func TestCastServiceMethodClonesNullableCollections(t *testing.T) {
	parsed := &model.Method{
		Name: "clone",
		Arguments: []*model.Argument{
			{Name: "items", Type: codegentest.NullableType(codegentest.ListType(codegentest.StringType()))},
		},
		ArgumentsData: &model.Data{
			Name: "CloneServiceCloneArguments",
			Members: []*model.DataMember{
				{Name: "items", Type: codegentest.NullableType(codegentest.ListType(codegentest.StringType()))},
			},
		},
		ResultType: codegentest.NullableType(codegentest.MapType(codegentest.StringType(), codegentest.StringType())),
	}

	method := castServiceMethod(&model.Service{Name: "CloneService"}, parsed)
	arguments := renderGoIRForTest(t, "goFunction", method.CloneArguments)
	result := renderGoIRForTest(t, "goFunction", method.CloneResult)

	for _, fragment := range []string{
		"if source.Items != nil {",
		"clonedValue0 := *source.Items",
		"clonedValue0 = make([]string, len((*source.Items)))",
		"cloned.Items = &clonedValue0",
	} {
		if !strings.Contains(arguments, fragment) {
			t.Fatalf("nullable list clone missing %q:\n%s", fragment, arguments)
		}
	}
	for _, fragment := range []string{
		"source := value.(*map[string]string)",
		"if source != nil {",
		"clonedValue2 = maps.Clone((*source))",
		"cloned = &clonedValue2",
	} {
		if !strings.Contains(result, fragment) {
			t.Fatalf("nullable map clone missing %q:\n%s", fragment, result)
		}
	}
}
