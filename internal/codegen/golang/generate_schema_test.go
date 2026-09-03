package golang_test

import (
	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/model"
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
	if !strings.Contains(string(goSchemaContent), `"AppConfig"`) ||
		!strings.Contains(string(goSchemaContent), `"demo.app.AppConfig"`) ||
		!strings.Contains(string(goSchemaContent), `Pub:       true`) {
		t.Fatalf("expected pub schema config pub flag, got:\n%s", string(goSchemaContent))
	}
	if !strings.Contains(string(goSchemaContent), `Via: skel.ActorViaClient`) {
		t.Fatalf("expected pub schema actor via, got:\n%s", string(goSchemaContent))
	}
}

// TestGeneratorGoSchemaHasNoBlankLineBeforeFields guards against stray blank
// lines being emitted before schema fields such as Hash and Type. Each
// conditional field block renders onto its own line, and a blank line at the
// composite-literal level would be preserved by gofmt, so the templates must
// trim leading whitespace around the optional deprecated-fields block.
func TestGeneratorGoSchemaHasNoBlankLineBeforeFields(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "skeled")

	userData := &model.Data{
		Pub:         true,
		Name:        "User",
		Description: "User record",
		Members: []*model.DataMember{
			{Name: "id", Type: stringTypeForTest()},
			{Name: "status", Sensitive: true, Type: scalarTypeForTest(model.ScalarInt)},
		},
	}

	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{
			userData,
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
		Enums: []*model.Enum{
			{Name: "Status", Items: []*model.EnumItem{{Name: "ACTIVE"}}},
		},
		Actors: []*model.Actor{
			{Pub: true, Name: "ClientActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)}},
		},
		Services: []*model.Service{
			{
				Pub:       true,
				Name:      "UserService",
				Audiences: []*model.ActorAudience{{Actor: "ClientActor", Via: string(model.ActorViaClient)}},
				Methods: []*model.Method{
					methodForTest("UserService", &model.Method{Name: "getUser", ResultType: dataTypeForTest(userData)}),
				},
			},
		},
		Tasks: []*model.Task{
			{
				Name: "RebuildTask",
				Triggers: []*model.TaskTrigger{
					triggerForTest("RebuildTask", &model.TaskTrigger{
						Name:      "atTime",
						Arguments: []*model.Argument{{Name: "startAt", Type: localDateTimeTypeForTest()}},
					}),
				},
			},
		},
		Events: []*model.Data{
			{
				Name: "UserCreated",
				Members: []*model.DataMember{
					{Name: "userId", Type: stringTypeForTest()},
				},
			},
		},
	})

	golang.Generate(pkg, golang.Option{Out: goOutDir})

	goSchemaContent := readFileForTest(t, filepath.Join(goOutDir, "schema.go"))
	assertNoBlankLineBeforeSchemaField(t, goSchemaContent, "Hash:")
	assertNoBlankLineBeforeSchemaField(t, goSchemaContent, "Type:")
	assertNoBlankLineBeforeSchemaField(t, goSchemaContent, "SkelName:")
}

// assertNoBlankLineBeforeSchemaField reports a failure when any schema field
// line (identified by fieldPrefix) is immediately preceded by a blank line.
func assertNoBlankLineBeforeSchemaField(t *testing.T, content string, fieldPrefix string) {
	t.Helper()
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, fieldPrefix) {
			continue
		}
		if i > 0 && strings.TrimSpace(lines[i-1]) == "" {
			t.Fatalf("unexpected blank line before schema field %q at line %d:\n%s", fieldPrefix, i+1, content)
		}
	}
}
