package golang_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/model"
)

func TestGeneratedCloneModuleCompilesAndIsolatesValues(t *testing.T) {
	generatedDir := filepath.Join(t.TempDir(), "generated")
	consumerDir := filepath.Join(filepath.Dir(generatedDir), "consumer")

	tItem := &model.TypeParameter{Name: "TItem"}
	child := &model.Data{
		Name: "Child",
		Members: []*model.DataMember{
			{Name: "name", Type: stringTypeForTest()},
			{Name: "content", Type: scalarTypeForTest(model.ScalarBinary)},
		},
	}
	page := &model.Data{
		Name:           "Page",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{
				Name: "items",
				Type: listTypeForTest(&model.Type{
					Kind:          model.TypeKindTypeParameter,
					TypeParameter: tItem,
				}),
			},
		},
	}
	payload := &model.Data{
		Name: "Payload",
		Members: []*model.DataMember{
			{Name: "content", Type: scalarTypeForTest(model.ScalarBinary)},
			{Name: "children", Type: listTypeForTest(dataTypeForTest(child))},
			{Name: "childrenByName", Type: mapTypeForTest(stringTypeForTest(), dataTypeForTest(child))},
			{Name: "optional", Type: nullableTypeForTest(dataTypeForTest(child))},
			{Name: "empty", Type: scalarTypeForTest(model.ScalarBinary)},
		},
	}
	envelope := &model.Data{
		Name: "Envelope",
		Members: []*model.DataMember{
			{Name: "page", Type: dataTypeForTest(page, dataTypeForTest(child))},
		},
	}
	service := &model.Service{
		Name: "CloneService",
		Methods: []*model.Method{
			methodForTest("CloneService", &model.Method{
				Name: "clone",
				Arguments: []*model.Argument{
					{Name: "payload", Type: dataTypeForTest(payload)},
				},
				ResultType: dataTypeForTest(envelope),
			}),
		},
	}
	domain := newModelDomainForTest(t, model.DomainSpec{
		Name:     "demo.clonefixture",
		Data:     []*model.Data{child, envelope, page, payload},
		Services: []*model.Service{service},
	})

	if err := golang.Generate(domain, golang.Option{
		Out:             generatedDir,
		AsModule:        true,
		Module:          "example.com/generated/clonefixture",
		CompilerVersion: "v0.11.1",
		VineVersion:     golang.DefaultVineVersion,
	}); err != nil {
		t.Fatalf("generate clone fixture module: %v", err)
	}

	if err := os.MkdirAll(consumerDir, 0o755); err != nil {
		t.Fatalf("create clone consumer directory: %v", err)
	}
	writeCloneConsumerFile(t, filepath.Join(consumerDir, "go.mod"), `module example.com/generated/cloneconsumer

go 1.26

require example.com/generated/clonefixture v0.0.0

replace example.com/generated/clonefixture => ../generated
`)
	writeCloneConsumerFile(t, filepath.Join(consumerDir, "clone_test.go"), `package cloneconsumer_test

import (
	"testing"

	fixture "example.com/generated/clonefixture"
)

func TestCloneValueIsolation(t *testing.T) {
	source := fixture.Payload{
		Content: []byte{1, 2},
		Children: []fixture.Child{
			{Name: "list", Content: []byte{3, 4}},
		},
		ChildrenByName: map[string]fixture.Child{
			"map": {Name: "map", Content: []byte{5, 6}},
		},
		Optional: &fixture.Child{Name: "optional", Content: []byte{7, 8}},
		Empty:    make([]byte, 0, 1<<20),
	}

	cloned := source.Clone()
	cloned.Content[0] = 11
	cloned.Children[0].Content[0] = 13
	mapped := cloned.ChildrenByName["map"]
	mapped.Content[0] = 15
	cloned.ChildrenByName["map"] = mapped
	cloned.Optional.Content[0] = 17

	if source.Content[0] != 1 {
		t.Fatalf("binary clone changed source: %v", source.Content)
	}
	if source.Children[0].Content[0] != 3 {
		t.Fatalf("list clone changed source: %v", source.Children)
	}
	if source.ChildrenByName["map"].Content[0] != 5 {
		t.Fatalf("map clone changed source: %v", source.ChildrenByName)
	}
	if source.Optional.Content[0] != 7 {
		t.Fatalf("nullable clone changed source: %v", source.Optional)
	}
	if cloned.Empty == nil || cap(cloned.Empty) != 0 {
		t.Fatalf("non-nil empty binary retained source storage: nil=%v cap=%d", cloned.Empty == nil, cap(cloned.Empty))
	}

	nilSource := fixture.Payload{}
	nilClone := nilSource.Clone()
	if nilClone.Content != nil || nilClone.Children != nil || nilClone.ChildrenByName != nil || nilClone.Optional != nil || nilClone.Empty != nil {
		t.Fatalf("clone did not preserve nil values: %+v", nilClone)
	}

	pageSource := fixture.Page[fixture.Child]{
		Items: []fixture.Child{{Name: "generic", Content: []byte{21, 22}}},
	}
	pageClone := pageSource.CloneBy(func(value fixture.Child) fixture.Child {
		return value.Clone()
	})
	pageClone.Items[0].Content[0] = 23
	if pageSource.Items[0].Content[0] != 21 {
		t.Fatalf("generic clone changed source: %v", pageSource.Items)
	}

	envelopeSource := fixture.Envelope{Page: pageSource}
	envelopeClone := envelopeSource.Clone()
	envelopeClone.Page.Items[0].Content[0] = 25
	if envelopeSource.Page.Items[0].Content[0] != 21 {
		t.Fatalf("nested generic clone changed source: %v", envelopeSource.Page.Items)
	}
}
`)

	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = consumerDir
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compile and test generated clone module: %v\n%s", err, output)
	}
}

func writeCloneConsumerFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write clone consumer file %s: %v", path, err)
	}
}
