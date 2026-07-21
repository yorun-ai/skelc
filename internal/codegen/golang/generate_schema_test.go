package golang_test

import (
	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorGoRendersSchemaFile(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "skeled")
	appContext := &model.Data{
		Pub:  true,
		Name: "AppContext",
		Members: []*model.DataMember{
			{Name: "name", Type: stringTypeForTest()},
		},
	}

	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.app",
		Data: []*model.Data{
			appContext,
		},
		Configs: []*model.Data{
			{
				Pub:       true,
				Name:      "AppConfig",
				Lifecycle: model.ConfigLifecycleEternal,
				Members: []*model.DataMember{
					{Name: "title", Type: stringTypeForTest()},
				},
			},
		},
		Actors: []*model.Actor{
			{Pub: true, Name: "ClientActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)}},
		},
		Services: []*model.Service{
			{
				Pub:       true,
				Name:      "AppService",
				Audiences: []*model.ActorAudience{{Actor: "ClientActor", Via: string(model.ActorViaClient)}},
				Methods: []*model.Method{
					methodForTest("AppService", &model.Method{Name: "getContext", ResultType: dataTypeForTest(appContext)}),
				},
			},
		},
	})

	pubOutDir := filepath.Join(t.TempDir(), "pub")
	golang.Generate(pkg, golang.Option{
		Out:          goOutDir,
		AsModule:     true,
		PubOut:       pubOutDir,
		ModulePrefix: "github.com/acme/skel",
	})

	goSchemaContent, err := os.ReadFile(filepath.Join(pubOutDir, "schema.go"))
	if err != nil {
		t.Fatalf("read go schema file: %v", err)
	}
	if !strings.Contains(string(goSchemaContent), "skel.RegisterDomainSchema(_DomainSchema)") {
		t.Fatalf("expected schema go registration, got:\n%s", string(goSchemaContent))
	}
	if !strings.Contains(string(goSchemaContent), `Name:     "AppContext"`) {
		t.Fatalf("expected pub schema data entry, got:\n%s", string(goSchemaContent))
	}
	if !strings.Contains(string(goSchemaContent), `Name:      "AppConfig"`) ||
		!strings.Contains(string(goSchemaContent), `SkelName:  "demo.app.AppConfig"`) ||
		!strings.Contains(string(goSchemaContent), `Pub:       true`) {
		t.Fatalf("expected pub schema config pub flag, got:\n%s", string(goSchemaContent))
	}
	if !strings.Contains(string(goSchemaContent), `Via: skel.ActorViaClient`) {
		t.Fatalf("expected pub schema actor via, got:\n%s", string(goSchemaContent))
	}
}
