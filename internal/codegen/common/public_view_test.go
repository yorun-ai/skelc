package common

import (
	"strings"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestBuildFiltersAndValidatesOneSharedPublicProjection(t *testing.T) {
	publicData := &model.Data{Pub: true, Name: "Public"}
	privateData := &model.Data{Name: "Private"}
	domain := model.NewDomainFromSpec(model.DomainSpec{
		Name: "demo.user", Data: []*model.Data{publicData, privateData},
		Enums: []*model.Enum{{Pub: true, Name: "Status"}, {Name: "InternalStatus"}},
	})
	view, err := BuildPublicView(domain)
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Data) != 1 || view.Data[0] != publicData || len(view.Enums) != 1 {
		t.Fatalf("unexpected public projection: %+v", view)
	}
}

func TestBuildValidatesPublicActorCredentialClosure(t *testing.T) {
	privateData := &model.Data{Name: "Secret", Kind: model.DataKindData}
	credential := &model.Data{Name: "Credential", Members: []*model.DataMember{{
		Name: "secret", Type: &model.Type{Kind: model.TypeKindData, Data: privateData},
	}}}
	domain := model.NewDomainFromSpec(model.DomainSpec{
		Name: "demo.user", Data: []*model.Data{privateData},
		Actors: []*model.Actor{{Pub: true, Name: "UserActor", AuthEnabled: true, AuthCredential: credential, AuthInfo: &model.Data{Name: "Info"}}},
	})
	_, err := BuildPublicView(domain)
	if err == nil || !strings.Contains(err.Error(), "pub actor UserActor credential references non-pub data Secret") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
