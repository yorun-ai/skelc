package hasher

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/model"
)

func buildEnumItemHashValues(items []*model.EnumItem) []*_EnumItemHashValue {
	values := make([]*_EnumItemHashValue, 0, len(items))
	for _, item := range items {
		values = append(values, &_EnumItemHashValue{
			Name:        item.Name,
			Description: item.Description,
		})
	}
	return values
}

func buildTypeParameterNames(items []*model.TypeParameter) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, item.Name)
	}
	return values
}

func buildActorViaNames(items []*model.ActorVia) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, string(item.Name))
	}
	return values
}

func (s *_hashState) buildMemberHashValues(items []*model.DataMember) []*_MemberHashValue {
	values := make([]*_MemberHashValue, 0, len(items))
	for _, item := range items {
		values = append(values, &_MemberHashValue{
			Name:        item.Name,
			Description: item.Description,
			Example:     item.Example,
			Type:        s.buildTypeHashValue(item.Type),
		})
	}
	return values
}

func (s *_hashState) buildArgumentHashValues(items []*model.Argument) []*_MemberHashValue {
	values := make([]*_MemberHashValue, 0, len(items))
	for _, item := range items {
		values = append(values, &_MemberHashValue{
			Name:        item.Name,
			Description: item.Description,
			Example:     item.Example,
			Type:        s.buildTypeHashValue(item.Type),
		})
	}
	return values
}

func (s *_hashState) buildTypeHashValue(typeMeta *model.Type) *_TypeHashValue {
	if typeMeta == nil {
		return nil
	}
	value := &_TypeHashValue{
		Kind:     typeKindName(typeMeta),
		Nullable: typeMeta.Nullable,
		Scalar:   scalarName(typeMeta),
		Name:     typeName(typeMeta),
		SkelName: typeMeta.SkelName,
	}
	switch typeMeta.Kind {
	case model.TypeKindEnum:
		if enum := s.enumBySkel[typeMeta.SkelName]; enum != nil {
			value.Hash = s.enumHash(enum)
		}
	case model.TypeKindData:
		if data := s.dataBySkel[typeMeta.SkelName]; data != nil {
			value.Hash = s.dataHash(data)
		}
	}
	if len(typeMeta.TypeArguments) > 0 {
		value.TypeArguments = make([]*_TypeHashValue, 0, len(typeMeta.TypeArguments))
		for _, typeArg := range typeMeta.TypeArguments {
			value.TypeArguments = append(value.TypeArguments, s.buildTypeHashValue(typeArg))
		}
	}
	if typeMeta.List != nil {
		value.Element = s.buildTypeHashValue(typeMeta.List.Value)
	}
	if typeMeta.Map != nil {
		value.Key = s.buildTypeHashValue(typeMeta.Map.Key)
		value.Value = s.buildTypeHashValue(typeMeta.Map.Value)
	}
	return value
}

func typeKindName(typeMeta *model.Type) string {
	switch typeMeta.Kind {
	case model.TypeKindScalar:
		return "scalar"
	case model.TypeKindList:
		return "list"
	case model.TypeKindMap:
		return "map"
	case model.TypeKindEnum:
		return "enum"
	case model.TypeKindData:
		return string(typeMeta.Data.Kind)
	case model.TypeKindTypeParameter:
		return "typeParameter"
	case model.TypeKindSkelPermissionCode:
		return "permissionCode"
	default:
		return fmt.Sprintf("unknown:%d", typeMeta.Kind)
	}
}

func scalarName(typeMeta *model.Type) string {
	if typeMeta.Kind != model.TypeKindScalar {
		return ""
	}
	return strings.ToLower(typeMeta.Scalar.Name())
}

func typeName(typeMeta *model.Type) string {
	switch typeMeta.Kind {
	case model.TypeKindScalar:
		return ""
	case model.TypeKindList, model.TypeKindMap:
		return ""
	case model.TypeKindSkelPermissionCode:
		return ""
	default:
		return typeMeta.Name()
	}
}
