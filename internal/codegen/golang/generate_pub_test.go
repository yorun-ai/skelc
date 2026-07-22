package golang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/model"
)

func TestGeneratorRendersPubGoView(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "skeled")
	goPubOutDir := filepath.Join(t.TempDir(), "skeledpub")

	userStatus := &model.Enum{Pub: true, Name: "UserStatus", Items: []*model.EnumItem{{Name: "ACTIVE"}}}
	unusedStatus := &model.Enum{Pub: true, Name: "UnusedStatus", Items: []*model.EnumItem{{Name: "IDLE"}}}
	publicStatus := &model.Enum{Pub: true, Name: "PublicStatus", Items: []*model.EnumItem{{Name: "READY"}}}
	address := &model.Data{
		Pub:  true,
		Name: "Address",
		Members: []*model.DataMember{
			{Name: "city", Type: stringTypeForTest()},
		},
	}
	user := &model.Data{
		Pub:  true,
		Name: "User",
		Members: []*model.DataMember{
			{Name: "status", Type: enumTypeForTest(userStatus)},
			{Name: "address", Type: dataTypeForTest(address)},
		},
	}
	unusedData := &model.Data{
		Pub:  true,
		Name: "UnusedData",
		Members: []*model.DataMember{
			{Name: "idle", Type: enumTypeForTest(unusedStatus)},
		},
	}
	partnerCredential := &model.Data{
		Name: "PartnerActorCredential",
		Members: []*model.DataMember{
			{Name: "subject", Type: stringTypeForTest()},
		},
	}
	partnerInfo := &model.Data{
		Name: "PartnerActorInfo",
		Members: []*model.DataMember{
			{Name: "userId", Type: stringTypeForTest()},
		},
	}
	publicCredential := &model.Data{
		Name: "PublicOnlyActorCredential",
		Members: []*model.DataMember{
			{Name: "subject", Type: stringTypeForTest()},
		},
	}
	publicInfo := &model.Data{
		Name: "PublicOnlyActorInfo",
		Members: []*model.DataMember{
			{Name: "userId", Type: stringTypeForTest()},
		},
	}
	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Actors: []*model.Actor{
			{Pub: true, Name: "OpenAPIActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaAgent)}},
			{Name: "PartnerActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)}, AuthEnabled: true, AuthCredential: partnerCredential, AuthInfo: partnerInfo},
			{Pub: true, Name: "PublicOnlyActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)}, AuthEnabled: true, AuthCredential: publicCredential, AuthInfo: publicInfo},
		},
		Enums: []*model.Enum{userStatus, unusedStatus, publicStatus},
		Data:  []*model.Data{address, user, unusedData},
		Configs: []*model.Data{
			{
				Pub:       true,
				Name:      "DemoConfig",
				Lifecycle: model.ConfigLifecycleEternal,
				Members: []*model.DataMember{
					{Name: "status", Type: enumTypeForTest(publicStatus)},
				},
			},
		},
		Services: []*model.Service{
			{
				Pub:       true,
				Name:      "UserService",
				Audiences: []*model.ActorAudience{{Actor: "OpenAPIActor"}},
				Methods: []*model.Method{
					methodForTest("UserService", &model.Method{Name: "getUser", ResultType: dataTypeForTest(user)}),
				},
			},
			{
				Name:      "PartnerService",
				Audiences: []*model.ActorAudience{{Actor: "PartnerActor"}},
				Methods: []*model.Method{
					methodForTest("PartnerService", &model.Method{Name: "ping", ResultType: stringTypeForTest()}),
				},
			},
		},
		Events: []*model.Data{
			{
				Pub:  true,
				Name: "UserCreatedEvent",
				Members: []*model.DataMember{
					{Name: "user", Type: dataTypeForTest(user)},
				},
			},
			{
				Name: "PartnerEvent",
				Members: []*model.DataMember{
					{Name: "message", Type: stringTypeForTest()},
				},
			},
		},
		Webs: []*model.Web{
			{Name: "UserPortalWeb", Audiences: []*model.ActorAudience{{Actor: "OpenAPIActor"}}},
		},
		Tasks: []*model.Task{
			{
				Name: "RebuildUserIndexTask",
				Triggers: []*model.TaskTrigger{
					triggerForTest("RebuildUserIndexTask", &model.TaskTrigger{
						Name: "atTime",
						Arguments: []*model.Argument{
							{Name: "startAt", Type: localDateTimeTypeForTest()},
						},
					}),
				},
			},
		},
	})

	golang.Generate(pkg, golang.Option{
		Out:          goOutDir,
		AsModule:     true,
		PubOut:       goPubOutDir,
		ModulePrefix: "github.com/acme/skel",
	})

	serviceContent, err := os.ReadFile(filepath.Join(goPubOutDir, "service.go"))
	if err != nil {
		t.Fatalf("read go service file: %v", err)
	}
	if !strings.Contains(string(serviceContent), "type UserServiceClient interface") {
		t.Fatalf("expected openapi service client, got:\n%s", string(serviceContent))
	}
	if strings.Contains(string(serviceContent), "type UserServiceServer interface") {
		t.Fatalf("did not expect pub-only codegen to render service server, got:\n%s", string(serviceContent))
	}
	if strings.Contains(string(serviceContent), "type PartnerServiceClient interface") {
		t.Fatalf("did not expect non-pub service in pub-only codegen, got:\n%s", string(serviceContent))
	}

	actorContent, err := os.ReadFile(filepath.Join(goPubOutDir, "actor.go"))
	if err != nil {
		t.Fatalf("read go actor file: %v", err)
	}
	if !strings.Contains(string(actorContent), "type OpenAPIActor struct {") {
		t.Fatalf("expected referenced pub actor, got:\n%s", string(actorContent))
	}
	if !strings.Contains(string(actorContent), "type PublicOnlyActor struct {") {
		t.Fatalf("expected explicitly pub actor, got:\n%s", string(actorContent))
	}
	if strings.Contains(string(actorContent), "type PartnerActor struct {") {
		t.Fatalf("did not expect non-pub actor in pub-only codegen, got:\n%s", string(actorContent))
	}
	if strings.Contains(string(actorContent), "PartnerActorAuthService") {
		t.Fatalf("did not expect non-pub actor auth service in pub-only codegen, got:\n%s", string(actorContent))
	}
	if !strings.Contains(string(actorContent), "rpc.Register(_PublicOnlyActorAuthServiceSpec)") {
		t.Fatalf("expected actor auth service to register, got:\n%s", string(actorContent))
	}
	if !strings.Contains(string(actorContent), "type PublicOnlyActorAuthServiceServer interface") {
		t.Fatalf("expected actor auth service server, got:\n%s", string(actorContent))
	}
	if !strings.Contains(string(actorContent), "Auth(credential PublicOnlyActorCredential) PublicOnlyActorInfo") {
		t.Fatalf("expected actor auth service auth method, got:\n%s", string(actorContent))
	}
	if !strings.Contains(string(actorContent), "type PublicOnlyActorCredential struct") || !strings.Contains(string(actorContent), "type PublicOnlyActorInfo struct") {
		t.Fatalf("expected actor credential/info data in actor.go, got:\n%s", string(actorContent))
	}
	if strings.Contains(string(actorContent), "type PublicOnlyActorAuthServiceClient interface") {
		t.Fatalf("did not expect actor auth service client, got:\n%s", string(actorContent))
	}
	if strings.Contains(string(actorContent), "type PublicOnlyActorAuthServiceClientER interface") {
		t.Fatalf("did not expect actor auth service er client, got:\n%s", string(actorContent))
	}

	dataContent, err := os.ReadFile(filepath.Join(goPubOutDir, "data.go"))
	if err != nil {
		t.Fatalf("read go data file: %v", err)
	}
	if strings.Contains(string(dataContent), "type PartnerActorCredential struct") || strings.Contains(string(dataContent), "type PartnerActorInfo struct") {
		t.Fatalf("did not expect non-pub actor credential/info data in pub-only go output, got:\n%s", string(dataContent))
	}
	if strings.Contains(string(dataContent), "type PublicOnlyActorCredential struct") || strings.Contains(string(dataContent), "type PublicOnlyActorInfo struct") {
		t.Fatalf("did not expect actor credential/info data in data.go, got:\n%s", string(dataContent))
	}

	eventContent, err := os.ReadFile(filepath.Join(goPubOutDir, "event.go"))
	if err != nil {
		t.Fatalf("read go event file: %v", err)
	}
	if !strings.Contains(string(eventContent), "type UserCreatedEventListener interface") {
		t.Fatalf("expected openapi event listener, got:\n%s", string(eventContent))
	}
	if strings.Contains(string(eventContent), "type UserCreatedEventEmitter interface") {
		t.Fatalf("did not expect pub-only codegen to render event emitter, got:\n%s", string(eventContent))
	}
	if strings.Contains(string(eventContent), "type PartnerEventListener interface") {
		t.Fatalf("did not expect non-pub event in pub-only codegen, got:\n%s", string(eventContent))
	}

	if !strings.Contains(string(dataContent), "type User struct") || !strings.Contains(string(dataContent), "type Address struct") {
		t.Fatalf("expected explicitly pub data, got:\n%s", string(dataContent))
	}
	if !strings.Contains(string(dataContent), "type UnusedData struct") {
		t.Fatalf("expected explicitly pub data in pub-only codegen, got:\n%s", string(dataContent))
	}

	enumContent, err := os.ReadFile(filepath.Join(goPubOutDir, "enum.go"))
	if err != nil {
		t.Fatalf("read go enum file: %v", err)
	}
	if !strings.Contains(string(enumContent), "type UserStatus string") {
		t.Fatalf("expected explicitly pub enum, got:\n%s", string(enumContent))
	}
	if !strings.Contains(string(enumContent), "type UnusedStatus string") {
		t.Fatalf("expected explicitly pub enum, got:\n%s", string(enumContent))
	}
	if !strings.Contains(string(enumContent), "type PublicStatus string") {
		t.Fatalf("expected explicitly pub enum, got:\n%s", string(enumContent))
	}

	configContent, err := os.ReadFile(filepath.Join(goPubOutDir, "config.go"))
	if err != nil {
		t.Fatalf("read go config file: %v", err)
	}
	if !strings.Contains(string(configContent), "type DemoConfig struct") {
		t.Fatalf("expected explicitly pub config, got:\n%s", string(configContent))
	}
	schemaContent, err := os.ReadFile(filepath.Join(goPubOutDir, "schema.go"))
	if err != nil {
		t.Fatalf("read go schema file: %v", err)
	}
	if !strings.Contains(string(schemaContent), `Domain: "demo.user"`) {
		t.Fatalf("expected pub schema file, got:\n%s", string(schemaContent))
	}
	assertFileMissing(t, filepath.Join(goPubOutDir, "task.go"))
}

func TestGeneratorRejectsImplicitPubDependencies(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "skeled")
	goPubOutDir := filepath.Join(t.TempDir(), "skeledpub")

	user := &model.Data{
		Name: "User",
		Members: []*model.DataMember{
			{Name: "name", Type: stringTypeForTest()},
		},
	}
	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Actors: []*model.Actor{
			{Pub: true, Name: "OpenAPIActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaAgent)}},
		},
		Data: []*model.Data{user},
		Services: []*model.Service{
			{
				Pub:       true,
				Name:      "UserService",
				Audiences: []*model.ActorAudience{{Actor: "OpenAPIActor"}},
				Methods: []*model.Method{
					methodForTest("UserService", &model.Method{Name: "getUser", ResultType: dataTypeForTest(user)}),
				},
			},
		},
	})

	err := golang.Generate(pkg, golang.Option{
		Out:          goOutDir,
		AsModule:     true,
		PubOut:       goPubOutDir,
		ModulePrefix: "github.com/acme/skel",
	})
	if err == nil || !strings.Contains(err.Error(), "pub service UserService.getUser references non-pub data User") {
		t.Fatalf("unexpected error: %v", err)
	}
}
