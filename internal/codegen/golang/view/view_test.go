package view

import (
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestViewSeparatesPubResources(t *testing.T) {
	domain := model.NewDomainFromSpec(model.DomainSpec{
		Name: "demo",
		Resources: []*model.Resource{
			{Pub: true, Name: "PublicUser", Actions: []*model.ResourceAction{{Name: "read"}}},
			{Name: "LocalUser", Actions: []*model.ResourceAction{{Name: "read"}}},
		},
	})

	pubView := New(ModePub, domain)
	if len(pubView.Resources) != 1 || pubView.Resources[0].Name != "PublicUser" {
		t.Fatalf("unexpected pub resources: %+v", pubView.Resources)
	}

	regularView := New(ModeRegular, domain)
	if len(regularView.Resources) != 1 || regularView.Resources[0].Name != "LocalUser" {
		t.Fatalf("unexpected regular resources: %+v", regularView.Resources)
	}
}

func TestPubViewRejectsServiceRequiringNonPubResource(t *testing.T) {
	domain := model.NewDomainFromSpec(model.DomainSpec{
		Name: "demo",
		Resources: []*model.Resource{
			{Name: "LocalUser", Actions: []*model.ResourceAction{{Name: "read"}}},
		},
		Services: []*model.Service{
			{
				Pub:      true,
				Name:     "UserService",
				SkelName: "demo.UserService",
				Require: &model.PermissionRequire{
					Expr: &model.PermissionExpr{
						Mode: model.PermissionRequireModeCode,
						Code: "demo.LocalUser:read",
					},
				},
			},
		},
	})

	defer func() {
		if recover() == nil {
			t.Fatalf("New() did not panic")
		}
	}()
	New(ModePub, domain)
}
