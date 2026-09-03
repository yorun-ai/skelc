package source

import (
	"testing"

	"go.yorun.ai/skelc/internal/model"
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

func TestCastMapTypeMapsUUIDKeyToSkelUUID(t *testing.T) {
	got := castType(&model.Type{
		Kind: model.TypeKindMap,
		Map: &model.MapType{
			Key:   &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarUUID},
			Value: &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarString},
		},
	})

	if got.Plain != "map[skel.UUID]string" {
		t.Fatalf("unexpected map type: %s", got.Plain)
	}
	if len(got.Imports) != 1 || got.Imports[0].Path != skelImport {
		t.Fatalf("unexpected map imports: %+v", got.Imports)
	}
}

func TestCastCollectionTypesUsePointersOnlyWhenNullable(t *testing.T) {
	tests := []struct {
		name        string
		type_       *model.Type
		wantPlain   string
		wantDefault string
	}{
		{name: "list", type_: listTypeForTest(stringTypeForTest()), wantPlain: "[]string", wantDefault: "[]string{}"},
		{name: "nullable list", type_: nullableTypeForTest(listTypeForTest(stringTypeForTest())), wantPlain: "*[]string", wantDefault: "nil"},
		{name: "map", type_: mapTypeForTest(stringTypeForTest(), stringTypeForTest()), wantPlain: "map[string]string", wantDefault: "map[string]string{}"},
		{name: "nullable map", type_: nullableTypeForTest(mapTypeForTest(stringTypeForTest(), stringTypeForTest())), wantPlain: "*map[string]string", wantDefault: "nil"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := castType(test.type_)
			if got.Plain != test.wantPlain || got.DefaultValue != test.wantDefault {
				t.Fatalf("unexpected collection type: plain=%q default=%q", got.Plain, got.DefaultValue)
			}
		})
	}
}
