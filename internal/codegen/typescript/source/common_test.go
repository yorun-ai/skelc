package source

import (
	"sort"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestCastTypeMapsBinaryToUint8Array(t *testing.T) {
	got := castType(&model.Type{
		Kind:   model.TypeKindScalar,
		Scalar: model.ScalarBinary,
	})
	if got.Plain != "Uint8Array" {
		t.Fatalf("unexpected binary type mapping: %s", got.Plain)
	}
}

func TestCastTypeMapsUUIDToString(t *testing.T) {
	got := castType(&model.Type{
		Kind:   model.TypeKindScalar,
		Scalar: model.ScalarUUID,
	})
	if got.Plain != "string" {
		t.Fatalf("unexpected uuid type mapping: %s", got.Plain)
	}
}

func TestCastTypeMapsJSONToString(t *testing.T) {
	got := castType(&model.Type{
		Kind:   model.TypeKindScalar,
		Scalar: model.ScalarJSON,
	})
	if got.Plain != "string" {
		t.Fatalf("unexpected json type mapping: %s", got.Plain)
	}
}

func TestCastTypeQualifiesExternalDataWithAlias(t *testing.T) {
	got := castType(&model.Type{
		Kind: model.TypeKindData,
		Data: &model.Data{
			Name: "UserSummary",
		},
		ExternalAlias: "userpub",
	})
	if got.Plain != "userpub.UserSummary" {
		t.Fatalf("unexpected external type: %s", got.Plain)
	}
}

func TestCastTypeQualifiesExternalEnumWithAlias(t *testing.T) {
	got := castType(&model.Type{
		Kind: model.TypeKindEnum,
		Enum: &model.Enum{
			Name: "UserStatus",
		},
		ExternalAlias: "userpub",
	})
	if got.Plain != "userpub.UserStatus" {
		t.Fatalf("unexpected external type: %s", got.Plain)
	}
}

func buildModelDomainForTest(t *testing.T, spec model.DomainSpec) *model.Domain {
	t.Helper()
	for _, enum := range spec.Enums {
		if enum.UnspecifiedItem == nil {
			enum.UnspecifiedItem = &model.EnumItem{Name: "UNSPECIFIED"}
		}
	}
	sort.Slice(spec.Enums, func(i, j int) bool { return spec.Enums[i].Name < spec.Enums[j].Name })
	sort.Slice(spec.Data, func(i, j int) bool { return spec.Data[i].Name < spec.Data[j].Name })
	sort.Slice(spec.Services, func(i, j int) bool { return spec.Services[i].Name < spec.Services[j].Name })

	return model.NewDomainFromSpec(spec)
}

func domainModelForTest(name string) model.DomainSpec {
	return model.DomainSpec{Name: name}
}

func stringTypeForTest() *model.Type {
	return scalarTypeForTest(model.ScalarString)
}

func intTypeForTest() *model.Type {
	return scalarTypeForTest(model.ScalarInt)
}

func binaryTypeForTest() *model.Type {
	return scalarTypeForTest(model.ScalarBinary)
}

func localDateTimeTypeForTest() *model.Type {
	return scalarTypeForTest(model.ScalarLocalDateTime)
}

func scalarTypeForTest(scalar model.Scalar) *model.Type {
	return &model.Type{Kind: model.TypeKindScalar, Scalar: scalar}
}

func listTypeForTest(value *model.Type) *model.Type {
	return &model.Type{Kind: model.TypeKindList, List: &model.ListType{Value: value}}
}

func mapTypeForTest(key *model.Type, value *model.Type) *model.Type {
	return &model.Type{Kind: model.TypeKindMap, Map: &model.MapType{Key: key, Value: value}}
}

func nullableTypeForTest(type_ *model.Type) *model.Type {
	type_.Nullable = true
	return type_
}

func dataTypeForTest(data *model.Data, typeArgs ...*model.Type) *model.Type {
	return &model.Type{
		Kind:          model.TypeKindData,
		Data:          data,
		SkelName:      data.SkelName,
		TypeArguments: typeArgs,
	}
}

func externalDataTypeForTest(data *model.Data, domainName string, alias string, explicitAlias bool) *model.Type {
	type_ := dataTypeForTest(data)
	type_.ExternalDomain = domainName
	type_.ExternalAlias = alias
	type_.ExternalAliasExplicit = explicitAlias
	return type_
}

func enumTypeForTest(enum *model.Enum) *model.Type {
	return &model.Type{
		Kind:     model.TypeKindEnum,
		Enum:     enum,
		SkelName: enum.SkelName,
	}
}

func typeParamForTest(name string) *model.TypeParameter {
	return &model.TypeParameter{Name: name}
}

func typeParamTypeForTest(typeParam *model.TypeParameter) *model.Type {
	return &model.Type{Kind: model.TypeKindTypeParameter, TypeParameter: typeParam}
}

func actorViaForTest(name model.ActorViaKind) *model.ActorVia {
	return &model.ActorVia{Name: string(name)}
}
