package analyzer

import (
	"testing"

	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

func TestParseData(t *testing.T) {
	data := parseDataTest(t, &grammar.Data{
		Decorators: []*grammar.Decorator{
			{Name: ident("desc"), Value: decoratorValue(`"Pagination information"`)},
		},
		Name: ident("Page"),
		TypeParameters: []*grammar.TypeParameter{
			{Name: ident("TItem")},
		},
		Members: []*grammar.DataMember{
			{
				Decorators: []*grammar.Decorator{
					{Name: ident("desc"), Value: decoratorValue(`"Data item"`)},
				},
				Name: ident("items"),
				Type: listType(refGrammarType("TItem")),
			},
			{
				Decorators: []*grammar.Decorator{
					{Name: ident("desc"), Value: decoratorValue(`"Next-page cursor"`)},
					{Name: ident("example"), Value: decoratorValue(`"cursor-1"`)},
				},
				Name: ident("nextToken"),
				Type: nullableType(plainType(grammar.String)),
			},
		},
	})

	if data.Name != "Page" {
		t.Fatalf("unexpected data name: %s", data.Name)
	}
	if data.Description != "Pagination information" {
		t.Fatalf("unexpected data description: %q", data.Description)
	}
	if !data.IsGeneric() {
		t.Fatal("data should be generic")
	}
	if len(data.TypeParameters) != 1 || data.TypeParameters[0].Name != "TItem" {
		t.Fatalf("unexpected type parameters: %+v", data.TypeParameters)
	}
	if len(data.Members) != 2 {
		t.Fatalf("unexpected member count: %d", len(data.Members))
	}
	if data.Members[0].Type.Kind != model.TypeKindList {
		t.Fatalf("unexpected first member type kind: %v", data.Members[0].Type.Kind)
	}
	if data.Members[0].Description != "Data item" {
		t.Fatalf("unexpected first member description: %q", data.Members[0].Description)
	}
	if data.Members[1].Example != `"cursor-1"` {
		t.Fatalf("unexpected second member example: %q", data.Members[1].Example)
	}
	if !data.Members[1].Type.Nullable {
		t.Fatal("expected nullable nextToken member")
	}
}

func TestParseDataReturnsErrorForDuplicatedMember(t *testing.T) {
	expectDataDiagnostic(t, "duplicated DataMember id", &grammar.Data{
		Name: ident("User"),
		Members: []*grammar.DataMember{
			{Name: ident("id"), Type: plainType(grammar.Int)},
			{Name: ident("id"), Type: plainType(grammar.String)},
		},
	})
}

func TestParseDataReturnsErrorForReservedKindSuffix(t *testing.T) {
	for _, name := range []string{"UserConfig", "UserEvent", "UserActor", "UserService", "UserWeb"} {
		t.Run(name, func(t *testing.T) {
			expectDataDiagnostic(t, "Data name must not end with", &grammar.Data{
				Name: ident(name),
			})
		})
	}
}

func TestParseDataReturnsErrorForNullableTypeParameter(t *testing.T) {
	expectDataDiagnostic(t, "TypeParameter TItem cannot be nullable", &grammar.Data{
		Name: ident("Page"),
		TypeParameters: []*grammar.TypeParameter{
			{Name: ident("TItem"), Nullable: true},
		},
	})
}

func TestParseDataReturnsErrorWhenMemberExampleHasNoDescription(t *testing.T) {
	expectDataDiagnostic(t, "decorator @example must be used with @desc", &grammar.Data{
		Name: ident("UserProfile"),
		Members: []*grammar.DataMember{
			{
				Decorators: []*grammar.Decorator{
					{Name: ident("example"), Value: decoratorValue(`"https://xxx.com/a.png"`)},
				},
				Name: ident("avatarUrl"),
				Type: nullableType(plainType(grammar.String)),
			},
		},
	})
}

func TestParseConfig(t *testing.T) {
	config := parseConfigTest(t, &grammar.Data{
		Decorators: []*grammar.Decorator{
			{Name: ident("desc"), Value: decoratorValue(`"Database configuration"`)},
		},
		Name:      ident("DatabaseConfig"),
		Qualifier: ident("eternal"),
		Members: []*grammar.DataMember{
			{
				Decorators: []*grammar.Decorator{
					{Name: ident("desc"), Value: decoratorValue(`"Connection address"`)},
				},
				Name: ident("connUrl"),
				Type: plainType(grammar.String),
			},
		},
	})
	if config.Kind != model.DataKindConfig {
		t.Fatalf("unexpected config kind: %v", config.Kind)
	}
	if config.Name != "DatabaseConfig" {
		t.Fatalf("unexpected config name: %s", config.Name)
	}
	if config.IsGeneric() {
		t.Fatal("config should not be generic")
	}
	if config.Lifecycle != model.ConfigLifecycleEternal {
		t.Fatalf("unexpected config lifecycle: %v", config.Lifecycle)
	}
}

func TestParseConfigReturnsErrorWhenNameDoesNotEndWithConfig(t *testing.T) {
	expectConfigDiagnostic(t, "Config name must end with Config", &grammar.Data{
		Name:      ident("Database"),
		Qualifier: ident("eternal"),
	})
}

func TestParseConfigReturnsErrorForTypeParameters(t *testing.T) {
	expectConfigDiagnostic(t, "does not support type parameters", &grammar.Data{
		Name:      ident("PageConfig"),
		Qualifier: ident("instant"),
		TypeParameters: []*grammar.TypeParameter{
			{Name: ident("TItem")},
		},
	})
}

func TestParseConfigReturnsErrorWithoutLifecycleQualifier(t *testing.T) {
	expectConfigDiagnostic(t, "requires lifecycle qualifier eternal/instant", &grammar.Data{
		Name: ident("DatabaseConfig"),
	})
}

func TestParseConfigReturnsErrorForInvalidLifecycleQualifier(t *testing.T) {
	expectConfigDiagnostic(t, "invalid lifecycle qualifier", &grammar.Data{
		Name:      ident("DatabaseConfig"),
		Qualifier: ident("cached"),
	})
}

func TestParseDataReturnsErrorForQualifier(t *testing.T) {
	expectDataDiagnostic(t, "does not support qualifier", &grammar.Data{
		Name:      ident("User"),
		Qualifier: ident("eternal"),
	})
}

func TestParseEvent(t *testing.T) {
	event := parseEventTest(t, &grammar.Event{
		Decorators: []*grammar.Decorator{
			{Name: ident("desc"), Value: decoratorValue(`"User created event"`)},
		},
		Pub:  true,
		Name: ident("UserCreatedEvent"),
		Payload: &grammar.EventPayload{
			Members: []*grammar.DataMember{
				{
					Decorators: []*grammar.Decorator{
						{Name: ident("desc"), Value: decoratorValue(`"User ID"`)},
					},
					Name: ident("userId"),
					Type: plainType(grammar.Int),
				},
			},
		},
	})
	if event.Kind != model.DataKindEvent {
		t.Fatalf("unexpected event kind: %v", event.Kind)
	}
	if !event.Pub {
		t.Fatal("expected pub event")
	}
}

func TestParseDataAllowsPub(t *testing.T) {
	data := parseDataTest(t, &grammar.Data{
		Pub:  true,
		Name: ident("User"),
	})
	if !data.Pub {
		t.Fatal("expected pub data")
	}
}

func TestParseConfigAllowsPub(t *testing.T) {
	config := parseConfigTest(t, &grammar.Data{
		Pub:       true,
		Name:      ident("DatabaseConfig"),
		Qualifier: ident("instant"),
	})
	if !config.Pub {
		t.Fatal("expected pub config")
	}
}
