package source

import (
	"strings"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestCastServiceMethodBuildsTypedDeepCloneHooks(t *testing.T) {
	payloadType := &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarBinary, Nullable: true}
	child := &model.Data{
		Name:    "Child",
		Members: []*model.DataMember{{Name: "name", Type: stringTypeForTest()}},
	}
	node := &model.Data{Name: "Node", Members: []*model.DataMember{
		{Name: "payload", Type: payloadType},
		{Name: "children", Type: listTypeForTest(dataTypeForTest(child))},
		{Name: "labels", Type: mapTypeForTest(stringTypeForTest(), stringTypeForTest())},
	}}
	parsed := &model.Method{
		Name: "clone",
		Arguments: []*model.Argument{
			{Name: "node", Type: nullableTypeForTest(dataTypeForTest(node))},
		},
		ArgumentsData: &model.Data{
			Name: "CloneServiceCloneArguments",
			Members: []*model.DataMember{
				{Name: "node", Type: nullableTypeForTest(dataTypeForTest(node))},
			},
		},
		ResultType: dataTypeForTest(node),
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

func TestCastServiceMethodBuildsCompatibleCloneForImportedData(t *testing.T) {
	external := &model.Data{Name: "User", Domain: "identity.user"}
	externalType := dataTypeForTest(external)
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
		t.Fatalf("imported data must use typed compatibility clone hooks: %+v", method)
	}
	for name, rendered := range map[string]string{
		"arguments": renderGoIRForTest(t, "goFunction", method.CloneArguments),
		"result":    renderGoIRForTest(t, "goFunction", method.CloneResult),
	} {
		for _, fragment := range []string{
			"any(value).(interface { Clone() userpub.User })",
			"vcode.MustUnmarshalJson[userpub.User](vcode.MustMarshalJson(value))",
		} {
			if !strings.Contains(rendered, fragment) {
				t.Fatalf("%s clone missing %q:\n%s", name, fragment, rendered)
			}
		}
	}
}

func TestCastServiceMethodBuildsCloneForRecursiveData(t *testing.T) {
	node := &model.Data{Name: "Node"}
	node.Members = []*model.DataMember{{Name: "children", Type: listTypeForTest(dataTypeForTest(node))}}
	parsed := &model.Method{
		Name:       "get",
		ResultType: dataTypeForTest(node),
	}

	method := castServiceMethod(&model.Service{Name: "NodeService"}, parsed)

	if method.CloneResult == nil {
		t.Fatalf("recursive data must use its generated Clone method: %+v", method)
	}
	if rendered := renderGoIRForTest(t, "goFunction", method.CloneResult); !strings.Contains(rendered, "cloned = source.Clone()") {
		t.Fatalf("recursive result clone missing Clone call:\n%s", rendered)
	}
}

func TestCastServiceMethodBuildsCompatibleCloneForImportedGenericData(t *testing.T) {
	tItem := &model.TypeParameter{Name: "TItem"}
	externalPage := &model.Data{
		Name:           "Page",
		Domain:         "identity.user",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: listTypeForTest(typeParamTypeForTest(tItem))},
		},
	}
	externalType := dataTypeForTest(externalPage, stringTypeForTest())
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
	externalType.ExternalAlias = "userpub"

	method := castServiceMethod(&model.Service{Name: "PageService"}, &model.Method{
		Name:       "list",
		ResultType: externalType,
	})

	if method.CloneResult == nil {
		t.Fatalf("imported generic data must use a typed compatibility clone hook: %+v", method)
	}
	rendered := renderGoIRForTest(t, "goFunction", method.CloneResult)
	for _, fragment := range []string{
		"interface { CloneBy(func(string) string) userpub.Page[string] }",
		"return cloner2.CloneBy(cloneImportedArgument1)",
		"cloned.Items[index4] = cloneImportedArgument1(value.Items[index4])",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("imported generic clone missing %q:\n%s", fragment, rendered)
		}
	}
	if strings.Contains(rendered, "vcode.") {
		t.Fatalf("generic compatibility clone must preserve typed callbacks instead of marshaling:\n%s", rendered)
	}
}

func TestCastServiceMethodMarshalsLegacyImportedGenericDataWithUnresolvedTransitiveType(t *testing.T) {
	tItem := &model.TypeParameter{Name: "TItem"}
	transitive := &model.Data{Name: "Meta", Domain: "shared.meta"}
	transitiveType := dataTypeForTest(transitive)
	transitiveType.ExternalDomain = "shared.meta"
	externalPage := &model.Data{
		Name:           "Page",
		Domain:         "identity.user",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: listTypeForTest(typeParamTypeForTest(tItem))},
			{Name: "metadata", Type: listTypeForTest(transitiveType)},
		},
	}
	externalType := dataTypeForTest(externalPage, stringTypeForTest())
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
	externalType.ExternalAlias = "userpub"

	method := castServiceMethod(&model.Service{Name: "PageService"}, &model.Method{
		Name:       "list",
		ResultType: externalType,
	})
	rendered := renderGoIRForTest(t, "goFunction", method.CloneResult)
	for _, fragment := range []string{
		"CloneBy(func(string) string) userpub.Page[string]",
		"cloned := vcode.MustUnmarshalJson[userpub.Page[string]](vcode.MustMarshalJson(value))",
		"cloned.Items[index4] = cloneImportedArgument1(value.Items[index4])",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("transitive compatibility clone missing %q:\n%s", fragment, rendered)
		}
	}
	if strings.Contains(rendered, "[]Meta") {
		t.Fatalf("unresolved transitive type leaked into generated clone code:\n%s", rendered)
	}
}

func TestCastServiceMethodBuildsGenericDataClone(t *testing.T) {
	tItem := &model.TypeParameter{Name: "TItem"}
	page := &model.Data{
		Name:           "Page",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: listTypeForTest(&model.Type{Kind: model.TypeKindTypeParameter, TypeParameter: tItem})},
		},
	}
	resultType := dataTypeForTest(page, stringTypeForTest())
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
