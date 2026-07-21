package source

import (
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestCastTypeUsesDefaultExternalPubPackageNameWithoutImportAlias(t *testing.T) {
	got := castType(&model.Type{
		Kind:               model.TypeKindData,
		Data:               &model.Data{Name: "UserSummary"},
		ExternalAlias:      "userpub",
		ExternalImportPath: "go.yorun.ai/app/vine/demo/user/userpub",
	})
	if got.Plain != "userpub.UserSummary" {
		t.Fatalf("unexpected external type: %s", got.Plain)
	}
	if len(got.Imports) != 1 {
		t.Fatalf("unexpected imports: %+v", got.Imports)
	}
	if got.Imports[0].Alias != "" {
		t.Fatalf("unexpected import alias: %s", got.Imports[0].Alias)
	}
}

func TestCastTypePreservesExplicitExternalImportAlias(t *testing.T) {
	got := castType(&model.Type{
		Kind:                  model.TypeKindData,
		Data:                  &model.Data{Name: "UserSummary"},
		ExternalAlias:         "account",
		ExternalAliasExplicit: true,
		ExternalImportPath:    "go.yorun.ai/app/vine/demo/user/userpub",
	})
	if got.Plain != "account.UserSummary" {
		t.Fatalf("unexpected external type: %s", got.Plain)
	}
	if len(got.Imports) != 1 {
		t.Fatalf("unexpected imports: %+v", got.Imports)
	}
	if got.Imports[0].Alias != "account" {
		t.Fatalf("unexpected import alias: %s", got.Imports[0].Alias)
	}
}

func TestCastEnumTypeUsesQualifiedUnspecifiedDefaultValue(t *testing.T) {
	got := castType(&model.Type{
		Kind: model.TypeKindEnum,
		Enum: &model.Enum{Name: "UserStatus", UnspecifiedItem: &model.EnumItem{Name: "UNSPECIFIED"}},
	})
	if got.DefaultValue != "UserStatusUnspecified" {
		t.Fatalf("unexpected default value: %s", got.DefaultValue)
	}
}

func TestCastExternalEnumTypeUsesQualifiedUnspecifiedDefaultValue(t *testing.T) {
	got := castType(&model.Type{
		Kind:               model.TypeKindEnum,
		Enum:               &model.Enum{Name: "UserStatus", UnspecifiedItem: &model.EnumItem{Name: "UNSPECIFIED"}},
		ExternalAlias:      "userpub",
		ExternalImportPath: "go.yorun.ai/app/vine/demo/user/userpub",
	})
	if got.DefaultValue != "userpub.UserStatusUnspecified" {
		t.Fatalf("unexpected default value: %s", got.DefaultValue)
	}
}
