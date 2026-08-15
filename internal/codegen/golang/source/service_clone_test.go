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

	for _, fragment := range []string{
		"source := value.(*_CloneServiceCloneArguments)",
		"clonedValue0 = (*source.Node).Clone()",
		"cloned.Node = &clonedValue",
	} {
		if !strings.Contains(method.CloneArguments, fragment) {
			t.Fatalf("CloneArguments missing %q:\n%s", fragment, method.CloneArguments)
		}
	}
	if !strings.Contains(method.CloneResult, "source := value.(Node)") ||
		!strings.Contains(method.CloneResult, "cloned = source.Clone()") {
		t.Fatalf("unexpected CloneResult:\n%s", method.CloneResult)
	}
	if got := importPaths(method.CloneImports); len(got) != 0 {
		t.Fatalf("unexpected clone imports: %v", got)
	}
}

func TestCastServiceMethodFallsBackForImportedData(t *testing.T) {
	external := &model.Data{Name: "User", Domain: "identity.user"}
	externalType := dataTypeForTest(external)
	externalType.ExternalDomain = "identity.user"
	externalType.ExternalImportPath = "example.com/identity"
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

	if method.CloneArguments != "" || method.CloneResult != "" {
		t.Fatalf("imported data must use Vine fallback: %+v", method)
	}
}

func TestCastServiceMethodFallsBackForRecursiveData(t *testing.T) {
	node := &model.Data{Name: "Node"}
	node.Members = []*model.DataMember{{Name: "children", Type: listTypeForTest(dataTypeForTest(node))}}
	parsed := &model.Method{
		Name:       "get",
		ResultType: dataTypeForTest(node),
	}

	method := castServiceMethod(&model.Service{Name: "NodeService"}, parsed)

	if method.CloneResult != "" {
		t.Fatalf("recursive data must use Vine fallback: %+v", method)
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

	if method.CloneResult == "" {
		t.Fatalf("expected generic result clone: %+v", method)
	}
	for _, fragment := range []string{
		"source.CloneBy(func(value string) string { return value })",
	} {
		if !strings.Contains(method.CloneResult, fragment) {
			t.Fatalf("generic clone hook missing %q:\n%s", fragment, method.CloneResult)
		}
	}
}
