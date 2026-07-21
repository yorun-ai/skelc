package analyzer

import (
	"testing"

	"go.yorun.ai/skelc/internal/parser/grammar"
)

func TestParseActor(t *testing.T) {
	actor := parseActor(&grammar.Actor{
		Decorators: []*grammar.Decorator{
			{Name: ident("desc"), Value: decoratorValue(`"Portal admin"`)},
		},
		Pub:  true,
		Name: ident("PortalAdminActor"),
		Vias: []*grammar.ActorVia{
			{Name: ident("client")},
			{Name: ident("openapi")},
		},
		Sections: []*grammar.ActorSection{
			grammarActorAuthSection(
				[]*grammar.DataMember{{Name: ident("subject"), Type: plainType(grammar.String)}},
				[]*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}},
			),
		},
	})
	if actor.Name != "PortalAdminActor" {
		t.Fatalf("unexpected actor name: %s", actor.Name)
	}
	if actor.SkelName != "" {
		t.Fatalf("unexpected actor skel name before domain normalize: %q", actor.SkelName)
	}
	if actor.Description != "Portal admin" {
		t.Fatalf("unexpected actor description: %q", actor.Description)
	}
	if !actor.Pub {
		t.Fatal("expected pub actor")
	}
	if len(actor.Vias) != 2 || actor.Vias[0].Name != "client" || actor.Vias[1].Name != "openapi" {
		t.Fatalf("unexpected actor vias: %+v", actor.Vias)
	}
	if actor.AuthCredential == nil || actor.AuthCredential.Name != "PortalAdminActorCredential" {
		t.Fatalf("unexpected actor credential: %+v", actor.AuthCredential)
	}
	if !actor.AuthCredential.Pub {
		t.Fatal("expected actor credential to follow actor pub")
	}
	if len(actor.AuthCredential.Members) != 1 || actor.AuthCredential.Members[0].Name != "subject" {
		t.Fatalf("unexpected actor credential members: %+v", actor.AuthCredential.Members)
	}
	if actor.AuthInfo == nil || actor.AuthInfo.Name != "PortalAdminActorInfo" {
		t.Fatalf("unexpected actor info: %+v", actor.AuthInfo)
	}
	if !actor.AuthInfo.Pub {
		t.Fatal("expected actor info to follow actor pub")
	}
	if len(actor.AuthInfo.Members) != 1 || actor.AuthInfo.Members[0].Name != "userId" {
		t.Fatalf("unexpected actor info members: %+v", actor.AuthInfo.Members)
	}
}

func TestParseActorRejectsNonStringCredentialMember(t *testing.T) {
	expectPanicContains(t, "actor credential member userId must be string", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{{Name: ident("client")}},
			Sections: []*grammar.ActorSection{
				grammarActorAuthSection(
					[]*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}},
					[]*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}},
				),
			},
		})
	})
}

func TestParseActorRejectsEmptyCredential(t *testing.T) {
	expectPanicContains(t, "actor credential must have at least one member", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{{Name: ident("client")}},
			Sections: []*grammar.ActorSection{
				grammarActorAuthSection([]*grammar.DataMember{}, []*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}}),
			},
		})
	})
}

func TestParseActorRejectsNullableCredentialMember(t *testing.T) {
	expectPanicContains(t, "actor credential member subject must be string", func() {
		credentialType := plainType(grammar.String)
		credentialType.Nullable = true
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{{Name: ident("client")}},
			Sections: []*grammar.ActorSection{
				grammarActorAuthSection(
					[]*grammar.DataMember{{Name: ident("subject"), Type: credentialType}},
					[]*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}},
				),
			},
		})
	})
}

func TestParseActorRejectsCredentialWithoutInfo(t *testing.T) {
	expectPanicContains(t, "auth must define credential and info together", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{{Name: ident("client")}},
			Sections: []*grammar.ActorSection{
				grammarActorAuthSection([]*grammar.DataMember{{Name: ident("subject"), Type: plainType(grammar.String)}}, nil),
			},
		})
	})
}

func TestParseActorRejectsInfoWithoutCredential(t *testing.T) {
	expectPanicContains(t, "auth must define credential and info together", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{{Name: ident("client")}},
			Sections: []*grammar.ActorSection{
				grammarActorAuthSection(nil, []*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}}),
			},
		})
	})
}

func TestParseActorRejectsDuplicatedAuth(t *testing.T) {
	expectPanicContains(t, "duplicated actor auth", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{{Name: ident("client")}},
			Sections: []*grammar.ActorSection{
				grammarActorAuthSection(
					[]*grammar.DataMember{{Name: ident("subject"), Type: plainType(grammar.String)}},
					[]*grammar.DataMember{{Name: ident("userId"), Type: plainType(grammar.Int)}},
				),
				grammarActorAuthSection(
					[]*grammar.DataMember{{Name: ident("tenant"), Type: plainType(grammar.String)}},
					[]*grammar.DataMember{{Name: ident("tenantId"), Type: plainType(grammar.Int)}},
				),
			},
		})
	})
}

func TestParseActorRequiresVia(t *testing.T) {
	expectPanicContains(t, "must have at least one via", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
		})
	})
}

func TestParseActorRejectsDuplicatedVia(t *testing.T) {
	expectPanicContains(t, "duplicated actor via client", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{
				{Name: ident("client")},
				{Name: ident("client")},
			},
		})
	})
}

func TestParseActorRejectsUnsupportedVia(t *testing.T) {
	expectPanicContains(t, "unexpected actor via partner", func() {
		parseActor(&grammar.Actor{
			Name: ident("PortalAdminActor"),
			Vias: []*grammar.ActorVia{{Name: ident("partner")}},
		})
	})
}

func grammarActorAuthSection(credentialMembers []*grammar.DataMember, infoMembers []*grammar.DataMember) *grammar.ActorSection {
	auth := &grammar.ActorAuth{}
	if credentialMembers != nil {
		auth.Credential = &grammar.ActorCredential{Members: credentialMembers}
	}
	if infoMembers != nil {
		auth.Info = &grammar.ActorInfo{Members: infoMembers}
	}
	return &grammar.ActorSection{Auth: auth}
}
