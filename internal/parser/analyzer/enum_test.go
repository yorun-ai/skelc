package analyzer

import (
	"testing"

	"go.yorun.ai/skelc/internal/parser/grammar"
)

func TestParseEnum(t *testing.T) {
	enum := parseEnum(&grammar.Enum{
		Decorators: []*grammar.Decorator{
			{Name: ident("desc"), Value: decoratorValue(`"User status"`)},
		},
		Pub:  true,
		Name: ident("UserStatus"),
		Items: []*grammar.EnumItem{
			{
				Decorators: []*grammar.Decorator{
					{Name: ident("desc"), Value: decoratorValue(`"Under review"`)},
				},
				Name: ident("PENDING"),
			},
			{Name: ident("ACTIVE")},
		},
	})

	if enum.Name != "UserStatus" {
		t.Fatalf("unexpected enum name: %s", enum.Name)
	}
	if enum.Description != "User status" {
		t.Fatalf("unexpected enum description: %q", enum.Description)
	}
	if !enum.Pub {
		t.Fatal("expected pub enum")
	}
	if enum.UnspecifiedItem == nil || enum.UnspecifiedItem.Name != unspecifiedEnumName {
		t.Fatalf("unexpected unspecified item: %+v", enum.UnspecifiedItem)
	}
	if len(enum.Items) != 2 {
		t.Fatalf("unexpected enum item count: %d", len(enum.Items))
	}
	if enum.Items[0].Name != "PENDING" || enum.Items[1].Name != "ACTIVE" {
		t.Fatalf("unexpected enum items: %+v", enum.Items)
	}
	if enum.Items[0].Description != "Under review" {
		t.Fatalf("unexpected enum item description: %q", enum.Items[0].Description)
	}
}

func TestParseEnumReturnsErrorForDuplicatedItem(t *testing.T) {
	expectPanicContains(t, "duplicated EnumItem ACTIVE", func() {
		parseEnum(&grammar.Enum{
			Name: ident("UserStatus"),
			Items: []*grammar.EnumItem{
				{Name: ident("ACTIVE")},
				{Name: ident("ACTIVE")},
			},
		})
	})
}

func TestParseEnumReturnsErrorForReservedKindSuffix(t *testing.T) {
	for _, name := range []string{"UserConfig", "UserEvent", "UserActor", "UserService", "UserWeb"} {
		t.Run(name, func(t *testing.T) {
			expectPanicContains(t, "Enum name must not end with", func() {
				parseEnum(&grammar.Enum{
					Name:  ident(name),
					Items: []*grammar.EnumItem{{Name: ident("ACTIVE")}},
				})
			})
		})
	}
}

func TestParseEnumReturnsErrorForReservedUnspecifiedItem(t *testing.T) {
	expectPanicContains(t, "reversed EnumItem value UNSPECIFIED", func() {
		parseEnum(&grammar.Enum{
			Name: ident("UserStatus"),
			Items: []*grammar.EnumItem{
				{Name: ident("UNSPECIFIED")},
			},
		})
	})
}
