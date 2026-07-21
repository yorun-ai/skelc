package hasher

import (
	"strings"

	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

func newHashTestDomain(serviceDescription string) *model.Domain {
	return analyzer.Analyze(&grammar.SkelContent{
		Domain: domainContent("demo.user"),
		Entries: []*grammar.SkelEntry{
			{
				Data: &grammar.Data{
					Name: ident("UserProfile"),
					Members: []*grammar.DataMember{
						{Name: ident("userId"), Type: plainType(grammar.String)},
					},
				},
			},
			{
				Actor: &grammar.Actor{
					Name: ident("ClientActor"),
					Vias: []*grammar.ActorVia{
						{Name: ident("client")},
					},
				},
			},
			{
				Service: &grammar.Service{
					Decorators: []*grammar.Decorator{
						{Name: ident("desc"), Value: &grammar.DecoratorValue{Raw: `"` + serviceDescription + `"`}},
					},
					Name:      ident("UserService"),
					Audiences: []*grammar.ServiceAudience{serviceAllow("ClientActor")},
					Methods: []*grammar.Method{
						{
							Name: ident("getUser"),
							Output: &grammar.MethodOutput{
								Type: refGrammarType("UserProfile"),
							},
						},
					},
				},
			},
		},
	}).Model()
}

func newHashActorCredentialTestDomain(credentialFieldName string) *model.Domain {
	return analyzer.Analyze(&grammar.SkelContent{
		Domain: domainContent("demo.user"),
		Entries: []*grammar.SkelEntry{
			{
				Actor: &grammar.Actor{
					Name: ident("ClientActor"),
					Vias: []*grammar.ActorVia{
						{Name: ident("client")},
					},
					Sections: []*grammar.ActorSection{
						actorAuthSection(
							[]*grammar.DataMember{{Name: ident(credentialFieldName), Type: plainType(grammar.String)}},
							[]*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}},
						),
					},
				},
			},
		},
	}).Model()
}

func newHashAllowViaTestDomain(via string) *model.Domain {
	return analyzer.Analyze(&grammar.SkelContent{
		Domain: domainContent("demo.user"),
		Entries: []*grammar.SkelEntry{
			{
				Actor: &grammar.Actor{
					Name: ident("ClientActor"),
					Vias: []*grammar.ActorVia{
						{Name: ident("client")},
						{Name: ident("openapi")},
					},
				},
			},
			{
				Service: &grammar.Service{
					Name:      ident("UserService"),
					Audiences: []*grammar.ServiceAudience{serviceAllowVia("ClientActor", via)},
					Methods: []*grammar.Method{
						{Name: ident("ping")},
					},
				},
			},
			{
				Web: &grammar.Web{
					Name:      ident("UserPortalWeb"),
					Audiences: []*grammar.WebAudience{webAllowVia("ClientActor", via)},
				},
			},
		},
	}).Model()
}

func domainContent(name string) *grammar.DomainContent {
	parts := strings.Split(name, ".")
	idents := make([]*grammar.Identifier, 0, len(parts))
	for _, part := range parts {
		idents = append(idents, ident(part))
	}
	return &grammar.DomainContent{
		Name: &grammar.QualifiedName{
			Parts: idents,
		},
	}
}

func actorAuthSection(credentialMembers []*grammar.DataMember, infoMembers []*grammar.DataMember) *grammar.ActorSection {
	return &grammar.ActorSection{
		Auth: &grammar.ActorAuth{
			Credential: &grammar.ActorCredential{Members: credentialMembers},
			Info:       &grammar.ActorInfo{Members: infoMembers},
		},
	}
}

func ident(value string) *grammar.Identifier {
	return &grammar.Identifier{Value: value}
}

func plainType(plainType grammar.PlainType) *grammar.Type {
	return &grammar.Type{Plain: &plainType}
}

func refGrammarType(name string, typeArgs ...*grammar.Type) *grammar.Type {
	return &grammar.Type{
		Reference: &grammar.ReferenceType{
			Name:          qualifiedName(name),
			TypeArguments: typeArgs,
		},
	}
}

func qualifiedName(name string) *grammar.QualifiedName {
	parts := strings.Split(name, ".")
	idents := make([]*grammar.Identifier, 0, len(parts))
	for _, part := range parts {
		idents = append(idents, ident(part))
	}
	return &grammar.QualifiedName{Parts: idents}
}

func serviceAllow(name string) *grammar.ServiceAudience {
	return &grammar.ServiceAudience{Keyword: "for", Actor: qualifiedName(name)}
}

func serviceAllowVia(name string, via string) *grammar.ServiceAudience {
	audience := serviceAllow(name)
	audience.Via = ident(via)
	return audience
}

func webAllowVia(name string, via string) *grammar.WebAudience {
	return &grammar.WebAudience{
		Keyword: "for",
		Actor:   qualifiedName(name),
		Via:     ident(via),
	}
}
