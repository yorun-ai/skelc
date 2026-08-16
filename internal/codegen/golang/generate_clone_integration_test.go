package golang_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/model"
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
	node := &model.Data{Name: "Node"}
	node.Members = []*model.DataMember{
		{Name: "children", Type: listTypeForTest(dataTypeForTest(node))},
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
		Data:     []*model.Data{child, envelope, node, page, payload},
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
	generatedData := readFileForTest(t, filepath.Join(generatedDir, "data.go"))
	for _, line := range []string{
		"// CloneBy returns a copy whose value isolation depends on each clone callback.",
		"// Passing an identity callback does not guarantee isolation for reference-backed values.",
	} {
		if !strings.Contains(generatedData, line) {
			t.Fatalf("generated CloneBy GoDoc missing %q:\n%s", line, generatedData)
		}
	}

	if err := os.MkdirAll(consumerDir, 0o755); err != nil {
		t.Fatalf("create clone consumer directory: %v", err)
	}
	writeCloneConsumerFile(t, filepath.Join(consumerDir, "go.mod"), `module example.com/generated/cloneconsumer

go 1.26.6

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

	nodeSource := fixture.Node{
		Children: []fixture.Node{{Children: []fixture.Node{{}}}},
	}
	nodeClone := nodeSource.Clone()
	nodeClone.Children[0].Children = append(nodeClone.Children[0].Children, fixture.Node{})
	if len(nodeSource.Children[0].Children) != 1 {
		t.Fatalf("recursive clone changed source: %+v", nodeSource)
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

func TestGeneratedCloneModuleUsesCurrentImportedCloneMethods(t *testing.T) {
	rootDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve current clone fixture directory: %v", err)
	}
	providerRegularDir := filepath.Join(rootDir, "provider")
	providerDir := filepath.Join(rootDir, "providerpub")
	consumerDir := filepath.Join(rootDir, "consumer")
	runnerDir := filepath.Join(rootDir, "runner")
	if err := os.MkdirAll(runnerDir, 0o755); err != nil {
		t.Fatalf("create current clone runner directory: %v", err)
	}

	tItem := &model.TypeParameter{Name: "TItem"}
	child := &model.Data{
		Pub:  true,
		Name: "Child",
		Members: []*model.DataMember{
			{Name: "content", Type: scalarTypeForTest(model.ScalarBinary)},
		},
	}
	page := &model.Data{
		Pub:            true,
		Name:           "Page",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: listTypeForTest(&model.Type{
				Kind:          model.TypeKindTypeParameter,
				TypeParameter: tItem,
			})},
		},
	}
	providerDomain := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.current",
		Data: []*model.Data{child, page},
	})
	if err := golang.Generate(providerDomain, golang.Option{
		Out:             providerRegularDir,
		PubOut:          providerDir,
		AsModule:        true,
		Module:          "example.com/generated/current",
		PubModule:       "example.com/generated/currentpub",
		CompilerVersion: "v0.12.0",
		VineVersion:     golang.DefaultVineVersion,
	}); err != nil {
		t.Fatalf("generate current clone provider modules: %v", err)
	}
	providerData := readFileForTest(t, filepath.Join(providerDir, "data.go"))
	for _, fragment := range []string{
		"func (v Child) Clone() Child",
		"func (v Page[TItem]) CloneBy(",
	} {
		if !strings.Contains(providerData, fragment) {
			t.Fatalf("current clone provider missing %q:\n%s", fragment, providerData)
		}
	}

	childType := externalDataTypeForTest(child, "demo.current")
	pageType := externalDataTypeForTest(page, "demo.current", childType)
	envelope := &model.Data{
		Name: "Envelope",
		Members: []*model.DataMember{
			{Name: "child", Type: childType},
			{Name: "page", Type: pageType},
		},
	}
	consumerDomain := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.currentconsumer",
		Imports: []*model.Import{{
			Name:   "demo.current",
			Alias:  "current",
			Domain: providerDomain,
		}},
		Data: []*model.Data{envelope},
	})
	if err := golang.Generate(consumerDomain, golang.Option{
		Out:             consumerDir,
		AsModule:        true,
		Module:          "example.com/generated/currentconsumer",
		CompilerVersion: "v0.12.0",
		VineVersion:     golang.DefaultVineVersion,
		Imports: map[string]string{
			"demo.current": "example.com/generated/currentpub",
		},
	}); err != nil {
		t.Fatalf("generate current clone consumer module: %v", err)
	}
	consumerData := readFileForTest(t, filepath.Join(consumerDir, "data.go"))
	for _, fragment := range []string{
		"Clone() currentpub.Child",
		"CloneBy(func(currentpub.Child) currentpub.Child) currentpub.Page[currentpub.Child]",
	} {
		if !strings.Contains(consumerData, fragment) {
			t.Fatalf("current clone consumer fast path missing %q:\n%s", fragment, consumerData)
		}
	}

	writeCloneConsumerFile(t, filepath.Join(runnerDir, "go.mod"), `module example.com/generated/currentrunner

go 1.26.6

require (
	example.com/generated/currentconsumer v0.0.0
	example.com/generated/currentpub v0.0.0
)

replace example.com/generated/currentconsumer => ../consumer

replace example.com/generated/currentpub => ../providerpub
`)
	writeCloneConsumerFile(t, filepath.Join(runnerDir, "clone_test.go"), `package currentrunner_test

import (
	"testing"

	consumer "example.com/generated/currentconsumer"
	current "example.com/generated/currentpub"
)

func TestCurrentImportedValueIsolation(t *testing.T) {
	source := consumer.Envelope{
		Child: current.Child{Content: []byte{1, 2}},
		Page: current.Page[current.Child]{
			Items: []current.Child{{Content: []byte{3, 4}}},
		},
	}
	cloned := source.Clone()
	cloned.Child.Content[0] = 11
	cloned.Page.Items[0].Content[0] = 13

	if source.Child.Content[0] != 1 {
		t.Fatalf("current non-generic clone changed source: %v", source.Child.Content)
	}
	if source.Page.Items[0].Content[0] != 3 {
		t.Fatalf("current generic clone changed source: %v", source.Page.Items)
	}
}
`)

	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = runnerDir
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compile and test current imported clone methods: %v\n%s", err, output)
	}
}

func TestGeneratedCloneModuleSupportsLegacyImportedData(t *testing.T) {
	rootDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve clone compatibility fixture directory: %v", err)
	}
	sharedDir := filepath.Join(rootDir, "shared")
	legacyDir := filepath.Join(rootDir, "legacy")
	generatedDir := filepath.Join(rootDir, "consumer")
	runnerDir := filepath.Join(rootDir, "runner")

	for _, directory := range []string{sharedDir, legacyDir, runnerDir} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create fixture directory %s: %v", directory, err)
		}
	}
	writeCloneConsumerFile(t, filepath.Join(sharedDir, "go.mod"), `module example.com/generated/shared

go 1.26
`)
	writeCloneConsumerFile(t, filepath.Join(sharedDir, "data.go"), `// Code generated by skelc v0.11.1. DO NOT EDIT.

package sharedpub

type Meta struct {
	Content []byte
}
`)
	writeCloneConsumerFile(t, filepath.Join(legacyDir, "go.mod"), `module example.com/generated/legacy

go 1.26

require example.com/generated/shared v0.0.0
`)
	writeCloneConsumerFile(t, filepath.Join(legacyDir, "data.go"), `// Code generated by skelc v0.11.1. DO NOT EDIT.

package legacypub

import shared "example.com/generated/shared"

type User struct {
	Content []byte
}

type Meta struct {
	Content []byte
}

type Page[TItem any] struct {
	Items []TItem
	Metas []Meta
	Shared []shared.Meta
}

func (v *Page[TItem]) Validate(path string) error {
	return nil
}
`)

	tItem := &model.TypeParameter{Name: "TItem"}
	legacyUser := &model.Data{
		Name: "User",
		Members: []*model.DataMember{
			{Name: "content", Type: scalarTypeForTest(model.ScalarBinary)},
		},
	}
	legacyMeta := &model.Data{
		Name: "Meta",
		Members: []*model.DataMember{
			{Name: "content", Type: scalarTypeForTest(model.ScalarBinary)},
		},
	}
	sharedMeta := &model.Data{
		Name: "Meta",
		Members: []*model.DataMember{
			{Name: "content", Type: scalarTypeForTest(model.ScalarBinary)},
		},
	}
	sharedDomain := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.shared",
		Data: []*model.Data{sharedMeta},
	})
	sharedMetaType := externalDataTypeForTest(sharedMeta, "demo.shared")
	legacyPage := &model.Data{
		Name:           "Page",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{
			{Name: "items", Type: listTypeForTest(&model.Type{
				Kind:          model.TypeKindTypeParameter,
				TypeParameter: tItem,
			})},
			{Name: "metas", Type: listTypeForTest(dataTypeForTest(legacyMeta))},
			{Name: "shared", Type: listTypeForTest(sharedMetaType)},
		},
	}
	legacyDomain := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.legacy",
		Imports: []*model.Import{{
			Name:   "demo.shared",
			Alias:  "shared",
			Domain: sharedDomain,
		}},
		Data: []*model.Data{legacyMeta, legacyPage, legacyUser},
	})
	legacyUserType := externalDataTypeForTest(legacyUser, "demo.legacy")
	legacyPageType := externalDataTypeForTest(legacyPage, "demo.legacy", legacyUserType)
	envelope := &model.Data{
		Name: "Envelope",
		Members: []*model.DataMember{
			{Name: "user", Type: legacyUserType},
			{Name: "page", Type: legacyPageType},
		},
	}
	consumerDomain := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.legacyconsumer",
		Imports: []*model.Import{{
			Name:   "demo.legacy",
			Alias:  "legacy",
			Domain: legacyDomain,
		}},
		Data: []*model.Data{envelope},
	})
	if err := golang.Generate(consumerDomain, golang.Option{
		Out:             generatedDir,
		AsModule:        true,
		Module:          "example.com/generated/legacyconsumer",
		CompilerVersion: "v0.12.0",
		VineVersion:     golang.DefaultVineVersion,
		Imports: map[string]string{
			"demo.legacy": "example.com/generated/legacy",
		},
	}); err != nil {
		t.Fatalf("generate legacy clone consumer module: %v", err)
	}

	generatedData := readFileForTest(t, filepath.Join(generatedDir, "data.go"))
	for _, fragment := range []string{
		"Clone() legacypub.User",
		"CloneBy(func(legacypub.User) legacypub.User) legacypub.Page[legacypub.User]",
		"vcode.MustMarshalCbor(value)",
	} {
		if !strings.Contains(generatedData, fragment) {
			t.Fatalf("legacy compatibility clone missing %q:\n%s", fragment, generatedData)
		}
	}

	writeCloneConsumerFile(t, filepath.Join(runnerDir, "go.mod"), `module example.com/generated/legacyrunner

go 1.26.6

require (
	example.com/generated/legacy v0.0.0
	example.com/generated/legacyconsumer v0.0.0
	example.com/generated/shared v0.0.0
)

replace example.com/generated/legacy => ../legacy

replace example.com/generated/legacyconsumer => ../consumer

replace example.com/generated/shared => ../shared
`)
	writeCloneConsumerFile(t, filepath.Join(runnerDir, "clone_test.go"), `package legacyrunner_test

import (
	"testing"

	legacy "example.com/generated/legacy"
	consumer "example.com/generated/legacyconsumer"
	shared "example.com/generated/shared"
)

func TestLegacyImportedValueIsolation(t *testing.T) {
	source := consumer.Envelope{
		User: legacy.User{Content: []byte{1, 2}},
		Page: legacy.Page[legacy.User]{
			Items: []legacy.User{{Content: []byte{3, 4}}},
			Metas: []legacy.Meta{{Content: []byte{5, 6}}},
			Shared: []shared.Meta{{Content: []byte{7, 8}}},
		},
	}
	cloned := source.Clone()
	cloned.User.Content[0] = 11
	cloned.Page.Items[0].Content[0] = 13
	cloned.Page.Metas[0].Content[0] = 15
	cloned.Page.Shared[0].Content[0] = 17

	if source.User.Content[0] != 1 {
		t.Fatalf("legacy non-generic clone changed source: %v", source.User.Content)
	}
	if source.Page.Items[0].Content[0] != 3 {
		t.Fatalf("legacy generic clone changed source: %v", source.Page.Items)
	}
	if source.Page.Metas[0].Content[0] != 5 {
		t.Fatalf("legacy nested data clone changed source: %v", source.Page.Metas)
	}
	if source.Page.Shared[0].Content[0] != 7 {
		t.Fatalf("legacy transitive data clone changed source: %v", source.Page.Shared)
	}
}
`)

	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = runnerDir
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("compile and test legacy imported clone compatibility: %v\n%s", err, output)
	}
}

func externalDataTypeForTest(data *model.Data, domain string, typeArgs ...*model.Type) *model.Type {
	type_ := dataTypeForTest(data, typeArgs...)
	type_.ExternalDomain = domain
	return type_
}

func writeCloneConsumerFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write clone consumer file %s: %v", path, err)
	}
}
