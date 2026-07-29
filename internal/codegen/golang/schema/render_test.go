package schema

import (
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenSchemaGoRendersActorAuthEnabled(t *testing.T) {
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Actors: []*model.Actor{
			{
				Name:        "ClientActor",
				Vias:        []*model.ActorVia{actorViaForTest(model.ActorViaClient)},
				AuthEnabled: true,
				AuthCredential: &model.Data{
					Name: "ClientActorCredential",
					Members: []*model.DataMember{
						{Name: "token", Type: stringTypeForTest()},
					},
				},
				AuthInfo: &model.Data{
					Name: "ClientActorInfo",
					Members: []*model.DataMember{
						{Name: "userId", Type: intTypeForTest()},
					},
				},
			},
			{
				Name: "AnonymousActor",
				Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)},
			},
		},
	})

	outputDir := filepath.Join(t.TempDir(), "skeled")
	gen := newGen(Option{
		Domain:      pkg,
		View:        mustView(t, view.ModeFull, pkg),
		Mode:        view.ModeFull,
		PackageName: "skeled",
		Out:         outputDir,
	})
	gen.gen()

	content, err := os.ReadFile(filepath.Join(outputDir, schemaGoFilename))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "AuthEnabled: true,") {
		t.Fatalf("expected generated schema to render AuthEnabled, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "AuthEnabled: false,") {
		t.Fatalf("expected generated schema to render disabled AuthEnabled, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), "PermEnabled: false,") {
		t.Fatalf("expected generated schema to render disabled PermEnabled, got:\n%s", string(content))
	}
}

func TestGenSchemaGoRendersDeprecatedFields(t *testing.T) {
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{{
			Name:             "User",
			Deprecated:       true,
			DeprecatedReason: "Use Profile instead",
			Members: []*model.DataMember{{
				Name:             "legacyId",
				Type:             stringTypeForTest(),
				Deprecated:       true,
				DeprecatedReason: "Use id instead",
			}},
		}},
	})
	outputDir := filepath.Join(t.TempDir(), "skeled")
	newGen(Option{
		Domain: pkg, View: mustView(t, view.ModeFull, pkg), Mode: view.ModeFull,
		PackageName: "skeled", Out: outputDir,
	}).gen()

	content, err := os.ReadFile(filepath.Join(outputDir, schemaGoFilename))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	generated := string(content)
	if strings.Count(generated, "Deprecated:") < 2 {
		t.Fatalf("expected data and member deprecation flags, got:\n%s", generated)
	}
	for _, reason := range []string{"Use Profile instead", "Use id instead"} {
		if !strings.Contains(generated, `"`+reason+`"`) {
			t.Fatalf("expected reason %q, got:\n%s", reason, generated)
		}
	}
}
