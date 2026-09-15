package source

import (
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/codegentest"
	"go.yorun.ai/skelc/internal/model"
)

func TestSpecTemplateRendersServiceSpecs(t *testing.T) {
	payload := &SpecTsPayload{Services: []*Service{{
		SkelName: "demo.template.DemoService",
		SpecName: "DemoServiceSpec",
		Methods: []*ServiceMethod{
			{Name: "getDemo", SkelName: "getDemo"},
			{Name: "ping", SkelName: "ping"},
		},
	}}}

	output := renderTemplate(t, specTsTemplate, payload)
	for _, check := range []string{
		"export const DemoServiceSpec = {",
		"serviceName: 'demo.template.DemoService'",
		"getDemo: 'getDemo'",
		"ping: 'ping'",
		"} as const;",
	} {
		if !strings.Contains(output, check) {
			t.Fatalf("expected rendered spec to contain %q, got:\n%s", check, output)
		}
	}
}

func TestSpecTemplateKeepsModuleSemanticsWhenEmpty(t *testing.T) {
	output := renderTemplate(t, specTsTemplate, &SpecTsPayload{})
	if !strings.Contains(output, "export {};") {
		t.Fatalf("expected rendered spec to keep module semantics, got:\n%s", output)
	}
}

func TestBuildSpecTsPayloadUsesFinalClientServiceSet(t *testing.T) {
	userActorDomain := buildModelDomainForTest(t, model.DomainSpec{
		Name:   "app",
		Actors: []*model.Actor{{Name: "UserActor", Vias: []*model.ActorVia{codegentest.ActorVia(model.ActorViaClient)}}},
	})
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Imports: []*model.Import{{
			Domain: userActorDomain,
			Name:   "app",
			Alias:  "app",
		}},
		Actors: []*model.Actor{{Name: "AgentActor", Vias: []*model.ActorVia{codegentest.ActorVia(model.ActorViaAgent)}}},
		Services: []*model.Service{
			{Name: "ExternalClientService", Audiences: []*model.ActorAudience{{Actor: "app.UserActor"}}, Methods: []*model.Method{{Name: "ping"}}},
			{Name: "BackendService", Pub: true, Methods: []*model.Method{{Name: "ping"}}},
		},
	})

	gen := newGen(pkg, ".")
	payload := gen.buildSpecTsPayload()
	if len(payload.Services) != 1 {
		t.Fatalf("unexpected service count: %d", len(payload.Services))
	}
	if got, want := payload.Services[0].SpecName, "ExternalClientServiceSpec"; got != want {
		t.Fatalf("unexpected service spec: got=%s want=%s", got, want)
	}
}

func TestBuildSpecTsPayloadRendersSparseWireForBinaryMethods(t *testing.T) {
	chunk := &model.Data{
		Name: "Chunk",
		Members: []*model.DataMember{{
			Name: "content",
			Type: codegentest.BinaryType(),
		}},
	}
	fileResult := &model.Data{
		Name: "FileResult",
		Members: []*model.DataMember{
			{
				Name: "content",
				Type: codegentest.NullableType(codegentest.BinaryType()),
			},
			{
				Name: "chunks",
				Type: codegentest.MapType(codegentest.IntType(), codegentest.DataType(chunk)),
			},
			{
				Name: "chunksById",
				Type: codegentest.MapType(codegentest.UUIDType(), codegentest.DataType(chunk)),
			},
		},
	}
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.file",
		Data: []*model.Data{chunk, fileResult},
		Services: []*model.Service{{
			Name:      "FileService",
			Audiences: []*model.ActorAudience{{Via: string(model.ActorViaClient)}},
			Methods: []*model.Method{
				{Name: "ping"},
				{
					Name: "upload",
					Arguments: []*model.Argument{{
						Name: "content",
						Type: codegentest.BinaryType(),
					}},
				},
				{
					Name:       "download",
					ResultType: codegentest.DataType(fileResult),
				},
			},
		}},
	})

	payload := newGen(pkg, ".").buildSpecTsPayload()
	output := renderTemplate(t, specTsTemplate, payload)
	for _, check := range []string{
		"import type { VrpcWireSchema } from '@yorun-ai/vrpc';",
		"createChunkWireSchema(): VrpcWireSchema",
		"createFileResultWireSchema(): VrpcWireSchema",
		"ping: 'ping'",
		"upload: {\n      arguments:",
		"download: {\n      result:",
		"kind: 'binary'",
		"nullable: true",
		"key: 'int'",
		"key: 'string'",
	} {
		if !strings.Contains(output, check) {
			t.Fatalf("expected rendered spec to contain %q, got:\n%s", check, output)
		}
	}
	if strings.Contains(output, "wire: {\n    ping:") {
		t.Fatalf("unexpected wire schema for a method without binary values:\n%s", output)
	}
}

func TestWireSchemaSupportsGenericAndRecursiveData(t *testing.T) {
	tItem := codegentest.TypeParam("TItem")
	wrapper := &model.Data{
		Name:           "Wrapper",
		SkelName:       "demo.file.Wrapper",
		TypeParameters: []*model.TypeParameter{tItem},
		Members: []*model.DataMember{{
			Name: "value",
			Type: codegentest.TypeParamType(tItem),
		}},
	}
	node := &model.Data{
		Name:     "Node",
		SkelName: "demo.file.Node",
	}
	node.Members = []*model.DataMember{
		{Name: "content", Type: codegentest.BinaryType()},
		{Name: "next", Type: codegentest.NullableType(codegentest.DataType(node))},
	}

	method := &model.Method{
		Name: "store",
		Arguments: []*model.Argument{
			{Name: "wrapped", Type: codegentest.DataType(wrapper, codegentest.BinaryType())},
			{Name: "node", Type: codegentest.DataType(node)},
		},
	}
	if !methodArgumentsContainBinary(method) {
		t.Fatal("expected generic Binary argument to select wire")
	}

	builder := newWireSchemaBuilder()
	builder.collectMethod(method)
	builder.prepareFactoryNames()
	factories := builder.renderFactories()
	var rendered strings.Builder
	for _, factory := range factories {
		rendered.WriteString(factory.Code)
		rendered.WriteString("\n")
	}
	code := rendered.String()
	for _, check := range []string{
		"function createWrapperWireSchema(\n  tItemWireSchema: VrpcWireSchema,",
		"value: tItemWireSchema",
		"createWrapperWireSchema({ kind: 'binary' })",
		"next: { ...createNodeWireSchema(), nullable: true }",
	} {
		if check == "createWrapperWireSchema({ kind: 'binary' })" {
			methodWire := builder.renderMethod(method)
			if !strings.Contains(methodWire.ArgumentsSchema, check) {
				t.Fatalf("expected method wire to contain %q, got:\n%s", check, methodWire.ArgumentsSchema)
			}
			continue
		}
		if !strings.Contains(code, check) {
			t.Fatalf("expected wire factories to contain %q, got:\n%s", check, code)
		}
	}
}
