package golang_test

import (
	"go.yorun.ai/skelc/internal/codegen/codegentest"
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
			{Name: "name", Type: codegentest.StringType()},
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
					{Name: "title", Type: codegentest.StringType()},
				},
			},
		},
		Actors: []*model.Actor{
			{Pub: true, Name: "ClientActor", Vias: []*model.ActorVia{codegentest.ActorVia(model.ActorViaClient)}},
		},
		Services: []*model.Service{
			{
				Pub:       true,
				Name:      "AppService",
				Audiences: []*model.ActorAudience{{Actor: "ClientActor", Via: string(model.ActorViaClient)}},
				Methods: []*model.Method{
					methodForTest("AppService", &model.Method{Name: "getContext", ResultType: codegentest.DataType(appContext)}),
				},
			},
		},
	})

	pubOutDir := filepath.Join(t.TempDir(), "pub")
	if err := generateFixture(pkg, golang.Option{
		Out:          goOutDir,
		AsModule:     true,
		PubOut:       pubOutDir,
		ModulePrefix: "github.com/acme/skel",
	}); err != nil {
		t.Fatal(err)
	}

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
			{Name: "id", Type: codegentest.StringType()},
			{Name: "status", Sensitive: true, Type: codegentest.ScalarType(model.ScalarInt)},
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
					{Name: "title", Type: codegentest.StringType()},
				},
			},
		},
		Enums: []*model.Enum{
			{Name: "Status", Items: []*model.EnumItem{{Name: "ACTIVE"}}},
		},
		Actors: []*model.Actor{
			{Pub: true, Name: "ClientActor", Vias: []*model.ActorVia{codegentest.ActorVia(model.ActorViaClient)}},
		},
		Services: []*model.Service{
			{
				Pub:       true,
				Name:      "UserService",
				Audiences: []*model.ActorAudience{{Actor: "ClientActor", Via: string(model.ActorViaClient)}},
				Methods: []*model.Method{
					methodForTest("UserService", &model.Method{Name: "getUser", ResultType: codegentest.DataType(userData)}),
				},
			},
		},
		Tasks: []*model.Task{
			{
				Name: "RebuildTask",
				Triggers: []*model.TaskTrigger{
					triggerForTest("RebuildTask", &model.TaskTrigger{
						Name:      "atTime",
						Arguments: []*model.Argument{{Name: "startAt", Type: codegentest.LocalDateTimeType()}},
					}),
				},
			},
		},
		Events: []*model.Data{
			{
				Name: "UserCreated",
				Members: []*model.DataMember{
					{Name: "userId", Type: codegentest.StringType()},
				},
			},
		},
	})

	if err := generateFixture(pkg, golang.Option{Out: goOutDir}); err != nil {
		t.Fatal(err)
	}

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

func TestGeneratorGoRendersWebMountInSpecAndSchema(t *testing.T) {
	for _, path := range []string{"", "/", "/portal/v1-assets/"} {
		pkg := newModelDomainForTest(t, model.DomainSpec{
			Name:   "demo.web",
			Actors: []*model.Actor{{Name: "ClientActor", Vias: []*model.ActorVia{codegentest.ActorVia(model.ActorViaClient)}}},
			Webs:   []*model.Web{{Name: "PortalWeb", MountPath: path, Audiences: []*model.ActorAudience{{Actor: "ClientActor"}}}},
		})
		out := filepath.Join(t.TempDir(), "skeled")
		if err := generateFixture(pkg, golang.Option{Out: out}); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"web.go", "schema.go"} {
			first := readFileForTest(t, filepath.Join(out, name))
			normalized := strings.Join(strings.Fields(first), " ")
			if path == "" {
				if strings.Contains(normalized, "MountPath:") {
					t.Fatalf("unspecified mount emitted in %s", name)
				}
			} else if !strings.Contains(normalized, `MountPath: "`+path+`",`) {
				t.Fatalf("mount path missing from %s:\n%s", name, first)
			}
			if err := generateFixture(pkg, golang.Option{Out: out}); err != nil {
				t.Fatal(err)
			}
			if second := readFileForTest(t, filepath.Join(out, name)); first != second {
				t.Fatal("non-deterministic generation")
			}
		}
	}
}
