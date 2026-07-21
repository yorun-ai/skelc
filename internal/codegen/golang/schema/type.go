package schema

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/model"
)

func (g *_Gen) buildMemberSchemas(members []*model.DataMember) []*_MemberSchema {
	schemas := make([]*_MemberSchema, 0, len(members))
	for _, member := range members {
		schemas = append(schemas, &_MemberSchema{
			Name: member.Name, Description: member.Description,
			Example: member.Example, Type: g.buildTypeSchema(member.Type),
		})
	}
	return schemas
}

func (g *_Gen) buildArgumentSchemas(arguments []*model.Argument) []*_MemberSchema {
	metas := make([]*_MemberSchema, 0, len(arguments))
	for _, argument := range arguments {
		metas = append(metas, &_MemberSchema{
			Name: argument.Name, Description: argument.Description,
			Example: argument.Example, Type: g.buildTypeSchema(argument.Type),
		})
	}
	return metas
}

func (g *_Gen) buildTypeSchema(type_ *model.Type) *_TypeSchema {
	if type_ == nil {
		return nil
	}
	meta := &_TypeSchema{Nullable: type_.Nullable}
	switch type_.Kind {
	case model.TypeKindScalar:
		meta.Kind = typeKindScalar
		meta.Scalar = scalar(type_.Scalar)
	case model.TypeKindSkelPermissionCode:
		meta.Kind = typeKindSkelPermissionCode
	case model.TypeKindList:
		meta.Kind = typeKindList
		meta.Element = g.buildTypeSchema(type_.List.Value)
	case model.TypeKindMap:
		meta.Kind = typeKindMap
		meta.Key = g.buildTypeSchema(type_.Map.Key)
		meta.Value = g.buildTypeSchema(type_.Map.Value)
	case model.TypeKindEnum:
		meta.Kind = typeKindEnum
		meta.Name = type_.Enum.Name
		meta.SkelName = type_.SkelName
	case model.TypeKindData:
		meta.Kind = g.typeMetaKind(type_.Data.Kind)
		meta.Name = type_.Data.Name
		meta.SkelName = type_.SkelName
		if len(type_.TypeArguments) > 0 {
			meta.TypeArguments = make([]*_TypeSchema, 0, len(type_.TypeArguments))
			for _, typeArgument := range type_.TypeArguments {
				meta.TypeArguments = append(meta.TypeArguments, g.buildTypeSchema(typeArgument))
			}
		}
	case model.TypeKindTypeParameter:
		meta.Kind = typeKindTypeParameter
		meta.Name = type_.TypeParameter.Name
	default:
		panic(fmt.Sprintf("unsupported type kind %d", type_.Kind))
	}
	return meta
}

func scalar(value model.Scalar) _Scalar {
	switch value {
	case model.ScalarInt:
		return scalarInt
	case model.ScalarFloat:
		return scalarFloat
	case model.ScalarBoolean:
		return scalarBool
	case model.ScalarString:
		return scalarString
	case model.ScalarDecimal:
		return scalarDecimal
	case model.ScalarBinary:
		return scalarBinary
	case model.ScalarTimestamp:
		return scalarTimestamp
	case model.ScalarDuration:
		return scalarDuration
	case model.ScalarLocalDate:
		return scalarLocalDate
	case model.ScalarLocalTime:
		return scalarLocalTime
	case model.ScalarLocalDateTime:
		return scalarLocalDateTime
	case model.ScalarUUID:
		return scalarUuid
	case model.ScalarJSON:
		return scalarJson
	default:
		panic(fmt.Sprintf("unsupported scalar %s", value.Name()))
	}
}

func (g *_Gen) typeMetaKind(kind model.DataKind) _TypeKind {
	switch kind {
	case model.DataKindData:
		return typeKindData
	case model.DataKindConfig:
		return typeKindConfig
	case model.DataKindEvent:
		return typeKindEvent
	default:
		panic("unexpected data kind")
	}
}

func (g *_Gen) skelName(name string) string {
	if alias, baseName, ok := strings.Cut(name, "."); ok {
		for _, import_ := range g.Domain.Imports() {
			if import_.Alias == alias {
				return fmt.Sprintf("%s.%s", import_.Name, baseName)
			}
		}
	}
	return fmt.Sprintf("%s.%s", g.Domain.Name(), name)
}
