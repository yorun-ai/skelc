package source

import (
	"testing"

	"go.yorun.ai/skelc/model"
)

func buildModelDomainForTest(t *testing.T, spec model.DomainSpec) *model.Domain {
	t.Helper()
	for _, data := range spec.Data {
		data.Kind = model.DataKindData
		data.Domain = spec.Name
	}
	for _, actor := range spec.Actors {
		if actor.AuthEnabled && actor.AuthService == nil {
			method := &model.Method{
				Name:       "auth",
				SkelName:   "auth",
				Auth:       model.AuthModeNoAuth,
				ResultType: dataTypeForTest(actor.AuthInfo),
				Arguments: []*model.Argument{
					{Name: "credential", Type: dataTypeForTest(actor.AuthCredential)},
				},
			}
			actor.AuthMethod = method
			actor.AuthService = &model.Service{
				Name:     actor.Name + "AuthService",
				SkelName: spec.Name + "." + actor.Name + "AuthService",
				Auth:     model.AuthModeNoAuth,
				Methods:  []*model.Method{method},
			}
		}
	}

	return model.NewDomainFromSpec(spec)
}

func domainModelForTest(name string) model.DomainSpec {
	return domainModelWithDescriptionForTest(name, "")
}

func domainModelWithDescriptionForTest(name string, description string) model.DomainSpec {
	return model.DomainSpec{
		Name:        name,
		Description: description,
	}
}

func stringTypeForTest() *model.Type {
	return scalarTypeForTest(model.ScalarString)
}

func intTypeForTest() *model.Type {
	return scalarTypeForTest(model.ScalarInt)
}

func localDateTimeTypeForTest() *model.Type {
	return scalarTypeForTest(model.ScalarLocalDateTime)
}

func scalarTypeForTest(scalar model.Scalar) *model.Type {
	return &model.Type{Kind: model.TypeKindScalar, Scalar: scalar}
}

func nullableTypeForTest(type_ *model.Type) *model.Type {
	type_.Nullable = true
	return type_
}

func listTypeForTest(value *model.Type) *model.Type {
	return &model.Type{Kind: model.TypeKindList, List: &model.ListType{Value: value}}
}

func mapTypeForTest(key *model.Type, value *model.Type) *model.Type {
	return &model.Type{Kind: model.TypeKindMap, Map: &model.MapType{Key: key, Value: value}}
}

func dataTypeForTest(data *model.Data, typeArgs ...*model.Type) *model.Type {
	return &model.Type{
		Kind:          model.TypeKindData,
		Data:          data,
		SkelName:      data.SkelName,
		TypeArguments: typeArgs,
	}
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

func importPaths(imports []*Import) []string {
	paths := make([]string, 0, len(imports))
	for _, import_ := range imports {
		paths = append(paths, import_.Path)
	}
	return paths
}
