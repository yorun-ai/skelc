package source

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

func TestNewGenDerivesPackageNameForApp(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("app"))

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"))

	if gen.pkgName != "@yorun-ai/skeled-app" {
		t.Fatalf("unexpected package name: %s", gen.pkgName)
	}
}

func TestNewGenDerivesPackageNameForAppDomain(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("sales.order"))

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"))

	if gen.pkgName != "@yorun-ai/skeled-sales-order" {
		t.Fatalf("unexpected package name: %s", gen.pkgName)
	}
}

func TestNewGenUsesTypeScriptModule(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("sales.order"))

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"), Option{
		Module: "@acme/orders",
	})

	if gen.pkgName != "@acme/orders" {
		t.Fatalf("unexpected package name: %s", gen.pkgName)
	}
}

func TestNewGenDerivesPackageNameFromTypeScriptModuleScope(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("sales.order"))

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"), Option{
		ModuleScope: "@acme/skeled",
	})

	if gen.pkgName != "@acme/skeled-sales-order" {
		t.Fatalf("unexpected package name: %s", gen.pkgName)
	}
}

func TestNewGenDerivesPackageNameFromNpmScope(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("sales.order"))

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"), Option{
		ModuleScope: "@acme",
	})

	if gen.pkgName != "@acme/sales-order" {
		t.Fatalf("unexpected package name: %s", gen.pkgName)
	}
}

func TestNewGenDerivesPubPackageNameFromTypeScriptModuleScope(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("sales.order"))

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"), Option{
		PubOnly:     true,
		ModuleScope: "@acme/skeled",
	})

	if gen.pkgName != "@acme/skeled-sales-orderpub" {
		t.Fatalf("unexpected package name: %s", gen.pkgName)
	}
}

func TestNewGenDerivesExternalTypeImportsFromTypeScriptModuleScope(t *testing.T) {
	userSummary := &model.Data{Name: "UserSummary", Pub: true}
	userDomain := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{userSummary},
	})
	order := &model.Data{
		Name: "Order",
		Members: []*model.DataMember{{
			Name: "buyer",
			Type: externalDataTypeForTest(userSummary, "demo.user", "user", true),
		}},
	}
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "sales.order",
		Imports: []*model.Import{{
			Domain:        userDomain,
			Name:          "demo.user",
			Alias:         "user",
			ExplicitAlias: true,
		}},
		Data: []*model.Data{order},
	})

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"), Option{
		ModuleScope: "@acme/skeled",
	})

	if gen.err != nil {
		t.Fatalf("unexpected generator error: %v", gen.err)
	}
	memberType := pkg.Data()[0].Members[0].Type
	if memberType.ExternalImportPath != "@acme/skeled-demo-userpub" {
		t.Fatalf("unexpected import path: %s", memberType.ExternalImportPath)
	}
	if memberType.ExternalAlias != "user" {
		t.Fatalf("unexpected import alias: %s", memberType.ExternalAlias)
	}
}

func TestNewGenDerivesPubExternalTypeImportsFromTypeScriptModuleScope(t *testing.T) {
	userSummary := &model.Data{Name: "UserSummary", Pub: true}
	userDomain := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{userSummary},
	})
	order := &model.Data{
		Name: "Order",
		Pub:  true,
		Members: []*model.DataMember{{
			Name: "buyer",
			Type: externalDataTypeForTest(userSummary, "demo.user", "user", true),
		}},
	}
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "sales.order",
		Imports: []*model.Import{{
			Domain:        userDomain,
			Name:          "demo.user",
			Alias:         "user",
			ExplicitAlias: true,
		}},
		Data: []*model.Data{order},
	})

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"), Option{
		PubOnly:     true,
		ModuleScope: "@acme/skeled",
	})

	if gen.err != nil {
		t.Fatalf("unexpected generator error: %v", gen.err)
	}
	memberType := pkg.Data()[0].Members[0].Type
	if memberType.ExternalImportPath != "@acme/skeled-demo-userpub" {
		t.Fatalf("unexpected import path: %s", memberType.ExternalImportPath)
	}
	if memberType.ExternalAlias != "user" {
		t.Fatalf("unexpected import alias: %s", memberType.ExternalAlias)
	}
}

func TestNewGenPubOnlyIgnoresInternalExternalTypeImports(t *testing.T) {
	userSummary := &model.Data{Name: "UserSummary", Pub: true}
	userDomain := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{userSummary},
	})
	internalOrder := &model.Data{
		Name: "InternalOrder",
		Members: []*model.DataMember{{
			Name: "buyer",
			Type: externalDataTypeForTest(userSummary, "demo.user", "user", true),
		}},
	}
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "sales.order",
		Imports: []*model.Import{{
			Domain:        userDomain,
			Name:          "demo.user",
			Alias:         "user",
			ExplicitAlias: true,
		}},
		Data: []*model.Data{internalOrder},
	})

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"), Option{PubOnly: true})

	if gen.err != nil {
		t.Fatalf("unexpected generator error: %v", gen.err)
	}
}

func TestRenderTsTrimsTrailingWhitespace(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "ts")
	pkg := buildModelDomainForTest(t, domainModelForTest("demo.user"))
	gen := newGen(pkg, outDir)

	gen.renderTs("sample.ts", "const value = 1;  \n\t\nconst next = 2;\t", nil)

	content, err := os.ReadFile(filepath.Join(outDir, "sample.ts"))
	if err != nil {
		t.Fatalf("read generated file failed: %v", err)
	}
	if got, want := string(content), "const value = 1;\n\nconst next = 2;\n"; got != want {
		t.Fatalf("unexpected content: got=%q want=%q", got, want)
	}
}

func TestClientServicesFiltersByActorVia(t *testing.T) {
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Actors: []*model.Actor{
			{Name: "ClientActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)}},
			{Name: "AgentActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaAgent)}},
			{Name: "OpenAPIActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaOpenAPI)}},
		},
		Services: []*model.Service{
			{Name: "ClientOnlyService", Audiences: []*model.ActorAudience{{Actor: "ClientActor"}}, Methods: []*model.Method{{Name: "ping"}}},
			{Name: "HybridService", Audiences: []*model.ActorAudience{{Actor: "AgentActor"}, {Actor: "ClientActor"}}, Methods: []*model.Method{{Name: "ping"}}},
			{Name: "AgentOnlyService", Audiences: []*model.ActorAudience{{Actor: "AgentActor"}}, Methods: []*model.Method{{Name: "ping"}}},
			{Name: "OpenAPIOnlyService", Audiences: []*model.ActorAudience{{Actor: "OpenAPIActor"}}, Methods: []*model.Method{{Name: "ping"}}},
			{Name: "ClientActorOpenAPIOnlyService", Audiences: []*model.ActorAudience{{Actor: "ClientActor", Via: string(model.ActorViaOpenAPI)}}, Methods: []*model.Method{{Name: "ping"}}},
			{Name: "ClientActorClientViaService", Audiences: []*model.ActorAudience{{Actor: "ClientActor", Via: string(model.ActorViaClient)}}, Methods: []*model.Method{{Name: "ping"}}},
			{Name: "InternalService", Methods: []*model.Method{{Name: "ping"}}},
		},
	})

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"))
	services := gen.clientServices()
	got := sliceutil.Map(services, func(service *model.Service) string { return service.Name })
	if want := []string{"ClientActorClientViaService", "ClientOnlyService", "HybridService"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected client services: got=%v want=%v", got, want)
	}
}

func TestClientServicesIncludesImportedClientActors(t *testing.T) {
	appDomain := buildModelDomainForTest(t, model.DomainSpec{
		Name:   "app",
		Actors: []*model.Actor{{Name: "UserActor", Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)}}},
	})
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Imports: []*model.Import{{
			Domain: appDomain,
			Name:   "app",
			Alias:  "app",
		}},
		Services: []*model.Service{{
			Name: "ImportedActorService", Audiences: []*model.ActorAudience{{Actor: "app.UserActor"}}, Methods: []*model.Method{{Name: "ping"}},
		}},
	})

	gen := newGen(pkg, filepath.Join(t.TempDir(), "ts"))
	services := gen.clientServices()
	got := sliceutil.Map(services, func(service *model.Service) string { return service.Name })
	if want := []string{"ImportedActorService"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected client services: got=%v want=%v", got, want)
	}
}

func TestGenRendersTypesWithoutClientServices(t *testing.T) {
	outDir := filepath.Join(t.TempDir(), "ts")
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Actors: []*model.Actor{{
			Name: "AgentActor",
			Vias: []*model.ActorVia{actorViaForTest(model.ActorViaAgent)},
		}},
		Services: []*model.Service{{
			Name: "AgentService", Audiences: []*model.ActorAudience{{Actor: "AgentActor"}}, Methods: []*model.Method{{Name: "ping"}},
		}},
	})

	gen := newGen(pkg, outDir)
	gen.generate()

	for _, filename := range []string{indexFilename, dataTsFilename, serviceTsFilename, specTsFilename} {
		if _, err := os.Stat(filepath.Join(outDir, filename)); err != nil {
			t.Fatalf("expected %s to exist: %v", filename, err)
		}
	}
	if _, err := os.Stat(filepath.Join(outDir, "package.json")); !os.IsNotExist(err) {
		t.Fatalf("expected package.json to be missing, err=%v", err)
	}
}
