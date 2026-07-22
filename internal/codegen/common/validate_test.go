package common

import (
	"strings"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestValidateDomainRejectsMalformedNestedModels(t *testing.T) {
	tests := []struct {
		name     string
		spec     model.DomainSpec
		expected string
	}{
		{name: "nil import", spec: model.DomainSpec{Imports: []*model.Import{nil}}, expected: "nil import"},
		{name: "missing imported domain", spec: model.DomainSpec{Imports: []*model.Import{{Name: "shared"}}}, expected: "has no domain model"},
		{name: "malformed imported domain", spec: model.DomainSpec{Imports: []*model.Import{{Name: "shared", Domain: model.NewDomainFromSpec(model.DomainSpec{Webs: []*model.Web{nil}})}}}, expected: "import shared"},
		{name: "incomplete actor auth", spec: model.DomainSpec{Actors: []*model.Actor{{Name: "Client", AuthEnabled: true}}}, expected: "incomplete auth support"},
		{name: "incomplete actor permission", spec: model.DomainSpec{Actors: []*model.Actor{{Name: "Client", PermEnabled: true}}}, expected: "incomplete permission support"},
		{name: "malformed optional permission service", spec: model.DomainSpec{Actors: []*model.Actor{{Name: "Client", PermService: &model.Service{Name: "Permission", Methods: []*model.Method{nil}}}}}, expected: "nil method"},
		{name: "nil resource action", spec: model.DomainSpec{Resources: []*model.Resource{{Name: "Document", Actions: []*model.ResourceAction{nil}}}}, expected: "nil action"},
		{name: "nil resource check", spec: model.DomainSpec{Resources: []*model.Resource{{Name: "Document", Checks: []*model.ResourceCheck{nil}}}}, expected: "nil check"},
		{name: "resource check without method", spec: model.DomainSpec{Resources: []*model.Resource{{Name: "Document", Checks: []*model.ResourceCheck{{Name: "owner"}}}}}, expected: "check owner is nil"},
		{name: "nil web", spec: model.DomainSpec{Webs: []*model.Web{nil}}, expected: "nil web"},
		{name: "nil web audience", spec: model.DomainSpec{Webs: []*model.Web{{Name: "Portal", Audiences: []*model.ActorAudience{nil}}}}, expected: "nil audience"},
		{name: "nil service audience", spec: model.DomainSpec{Services: []*model.Service{{Name: "Documents", Audiences: []*model.ActorAudience{nil}}}}, expected: "nil audience"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			domain := model.NewDomainFromSpec(test.spec)
			err := ValidateDomain(domain)
			if err == nil || !strings.Contains(err.Error(), test.expected) {
				t.Fatalf("expected error containing %q, got %v", test.expected, err)
			}
		})
	}
}
