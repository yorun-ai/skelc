package schema

import (
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/model"
	"path/filepath"
	"testing"
)

func TestBuildDomainSchemaCopiesHashes(t *testing.T) {
	userProfile := &model.Data{
		Name: "UserProfile",
		Members: []*model.DataMember{
			{Name: "userId", Type: stringTypeForTest()},
		},
	}
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name:        "demo.user",
		Description: "User domain",
		Data:        []*model.Data{userProfile},
		Actors: []*model.Actor{{
			Name: "ClientActor",
			Vias: []*model.ActorVia{actorViaForTest(model.ActorViaClient)},
		}},
		Services: []*model.Service{{
			Name:      "UserService",
			Audiences: []*model.ActorAudience{{Actor: "ClientActor", Via: string(model.ActorViaClient)}},
			Methods: []*model.Method{{
				Name:       "getUser",
				ResultType: dataTypeForTest(userProfile),
			}},
		}},
	})
	fillModelHashesForTest(pkg)

	gen := newGen(Option{
		CompilerVersion: "v1.2.3",
		Domain:          pkg,
		View:            mustView(t, view.ModeFull, pkg),
		Mode:            view.ModeFull,
		PackageName:     "skeled",
		Out:             filepath.Join(t.TempDir(), "skeled"),
	})
	meta := gen.buildDomainSchema()

	if meta.Generated == nil || meta.Generated.CompilerVersion != "v1.2.3" {
		t.Fatalf("unexpected generated info: %+v", meta.Generated)
	}
	if meta.Hash != "domain-hash" {
		t.Fatalf("unexpected domain hash: %q", meta.Hash)
	}
	if !meta.Full {
		t.Fatal("expected full domain schema")
	}
	if len(meta.Data) != 1 || meta.Data[0].Hash != "data-hash" {
		t.Fatalf("unexpected data hash: %+v", meta.Data)
	}
	if len(meta.Actors) != 1 || meta.Actors[0].Hash != "actor-hash" {
		t.Fatalf("unexpected actor hash: %+v", meta.Actors)
	}
	if meta.Actors[0].AuthEnabled {
		t.Fatal("expected actor auth disabled")
	}
	if len(meta.Services) != 1 || meta.Services[0].Hash != "service-hash" {
		t.Fatalf("unexpected service hash: %+v", meta.Services)
	}
	if len(meta.Services[0].Audiences) != 1 || string(meta.Services[0].Audiences[0].Via) != string(model.ActorViaClient) {
		t.Fatalf("unexpected service for via: %+v", meta.Services[0].Audiences)
	}
	if len(meta.Services[0].Methods) != 1 || meta.Services[0].Methods[0].Hash != "method-hash" {
		t.Fatalf("unexpected method hash: %+v", meta.Services[0].Methods)
	}
	if string(meta.Services[0].AuthMode) != string(model.AuthModeUnset) {
		t.Fatalf("expected service auth unset, got %s", meta.Services[0].AuthMode)
	}
	if string(meta.Services[0].Methods[0].AuthMode) != string(model.AuthModeUnset) {
		t.Fatalf("expected method auth unset, got %s", meta.Services[0].Methods[0].AuthMode)
	}
}

func TestBuildDomainSchemaSplitFullFlagAndContent(t *testing.T) {
	pubData := &model.Data{
		Pub:  true,
		Name: "PubData",
		Members: []*model.DataMember{
			{Name: "id", Type: stringTypeForTest()},
		},
	}
	regularData := &model.Data{
		Name: "RegularData",
		Members: []*model.DataMember{
			{Name: "id", Type: stringTypeForTest()},
		},
	}
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{pubData, regularData},
		Services: []*model.Service{
			{
				Pub:  true,
				Name: "PubService",
				Methods: []*model.Method{
					{Name: "getPub", ResultType: dataTypeForTest(pubData)},
				},
			},
			{
				Name: "RegularService",
				Methods: []*model.Method{
					{Name: "getRegular", ResultType: dataTypeForTest(regularData)},
				},
			},
		},
	})

	pubGen := newGen(Option{
		Domain:      pkg,
		View:        mustView(t, view.ModePub, pkg),
		Mode:        view.ModePub,
		PackageName: "userpub",
		Out:         filepath.Join(t.TempDir(), "pub"),
	})
	pubSchema := pubGen.buildDomainSchema()
	if pubSchema.Full {
		t.Fatal("did not expect pub schema to be full")
	}
	if len(pubSchema.Data) != 1 || pubSchema.Data[0].Name != "PubData" {
		t.Fatalf("unexpected pub schema data: %+v", pubSchema.Data)
	}
	if len(pubSchema.Services) != 1 || pubSchema.Services[0].Name != "PubService" {
		t.Fatalf("unexpected pub schema services: %+v", pubSchema.Services)
	}

	regularGen := newGen(Option{
		Domain:      pkg,
		View:        mustView(t, view.ModeRegular, pkg),
		Mode:        view.ModeRegular,
		PackageName: "user",
		Out:         filepath.Join(t.TempDir(), "regular"),
	})
	regularSchema := regularGen.buildDomainSchema()
	if !regularSchema.Full {
		t.Fatal("expected regular schema to be full")
	}
	if len(regularSchema.Data) != 2 {
		t.Fatalf("expected regular schema to include full data, got %+v", regularSchema.Data)
	}
	if len(regularSchema.Services) != 2 {
		t.Fatalf("expected regular schema to include full services, got %+v", regularSchema.Services)
	}
}

func TestBuildDomainSchemaConfigLifecycleUsesConfValue(t *testing.T) {
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Configs: []*model.Data{{
			Pub:       true,
			Name:      "UserConfig",
			Lifecycle: model.ConfigLifecycleEternal,
			Members: []*model.DataMember{
				{Name: "pageSize", Type: intTypeForTest()},
			},
		}},
	})

	gen := newGen(Option{
		Domain:      pkg,
		View:        mustView(t, view.ModeFull, pkg),
		Mode:        view.ModeFull,
		PackageName: "skeled",
		Out:         filepath.Join(t.TempDir(), "skeled"),
	})
	meta := gen.buildDomainSchema()

	if len(meta.Configs) != 1 {
		t.Fatalf("expected one config, got %d", len(meta.Configs))
	}
	if meta.Configs[0].Lifecycle != "ETERNAL" {
		t.Fatalf("unexpected config lifecycle: %s", meta.Configs[0].Lifecycle)
	}
	if !meta.Configs[0].Pub {
		t.Fatal("expected config pub flag")
	}
}

func TestBuildDomainSchemaIncludesActorAuthMethod(t *testing.T) {
	pkg := buildModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Actors: []*model.Actor{{
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
					{Name: "userId", Type: stringTypeForTest()},
				},
			},
		}},
	})
	pkg.Actors()[0].AuthService.Hash = "auth-service-hash"
	pkg.Actors()[0].AuthMethod.Hash = "auth-method-hash"

	gen := newGen(Option{
		Domain:      pkg,
		View:        mustView(t, view.ModeFull, pkg),
		Mode:        view.ModeFull,
		PackageName: "skeled",
		Out:         filepath.Join(t.TempDir(), "skeled"),
	})
	meta := gen.buildDomainSchema()

	if len(meta.Actors) != 1 {
		t.Fatalf("expected one actor, got %d", len(meta.Actors))
	}
	actor := meta.Actors[0]
	if !actor.AuthEnabled {
		t.Fatal("expected actor auth enabled")
	}
	if actor.AuthMethod == nil {
		t.Fatal("expected actor auth method")
	}
	if actor.AuthMethod.SkelName != "auth" || actor.AuthMethod.Hash != "auth-method-hash" {
		t.Fatalf("unexpected actor auth method: %+v", actor.AuthMethod)
	}
	if actor.AuthService == nil || len(actor.AuthService.Methods) != 1 || actor.AuthService.Methods[0].SkelName != actor.AuthMethod.SkelName {
		t.Fatalf("unexpected actor auth service: %+v", actor.AuthService)
	}
}
