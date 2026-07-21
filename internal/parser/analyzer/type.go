package analyzer

import (
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

const (
	typeKindReference model.TypeKind = -1
	typeKindNone      model.TypeKind = 0
)

const scalarNone model.Scalar = 0

var (
	mapKeyTypes       = []model.TypeKind{model.TypeKindScalar, model.TypeKindEnum}
	mapKeyScalarTypes = []model.Scalar{model.ScalarInt, model.ScalarString}
)

type refContext struct {
	enums          map[string]*model.Enum
	dataList       map[string]*model.Data
	typeParameters map[string]*model.TypeParameter
	imports        map[string]*domainImport
}

func fixTypeRef(t *model.Type, refCtx *refContext) {
	// 1. fix Enum/Data/TypeParameter type references
	// 2. check map key type (int/string/Enum)

	if t == nil {
		return
	}

	switch t.Kind {
	case typeKindReference:
		refName := t.SkelName
		refQualifier := t.ExternalAlias
		if refQualifier != "" {
			import_ := refCtx.imports[refQualifier]
			checkutil.CheckNotNil(import_, "%s import alias %s not found", t.Pos, refQualifier)
			enum, enumOK := import_.Domain.enumsMap[refName]
			dataType, dataOK := import_.Domain.dataMap[refName]
			checkutil.Check(enumOK || dataOK, "%s definition of %s.%s not found", t.Pos, refQualifier, refName)
			if enumOK {
				checkutil.Check(enum.Pub, "%s imported enum %s.%s is not public", t.Pos, import_.Model.Alias, refName)
				t.Kind = model.TypeKindEnum
				t.Enum = enum
				t.SkelName = enum.SkelName
				t.ExternalDomain = import_.Domain.name
				t.ExternalAlias = import_.Model.Alias
				t.ExternalAliasExplicit = import_.Model.ExplicitAlias
				return
			}
			checkutil.Check(dataType.Pub, "%s imported data %s.%s is not public", t.Pos, import_.Model.Alias, refName)
			t.Kind = model.TypeKindData
			t.Data = dataType
			t.SkelName = dataType.SkelName
			t.ExternalAlias = import_.Model.Alias
			t.ExternalDomain = import_.Domain.name
			t.ExternalAliasExplicit = import_.Model.ExplicitAlias
			for _, typeArg := range t.TypeArguments {
				fixTypeRef(typeArg, refCtx)
			}
			checkTypeArguments(t, refName)
			return
		}
		enum, enumOK := refCtx.enums[refName]
		dataType, dataOK := refCtx.dataList[refName]
		param, paramOK := refCtx.typeParameters[refName]
		checkutil.Check(enumOK || dataOK || paramOK, "%s definition of %s not found", t.Pos, refName)
		if enumOK {
			t.Kind = model.TypeKindEnum
			t.Enum = enum
			t.SkelName = enum.SkelName
			break
		}
		if dataOK {
			t.Kind = model.TypeKindData
			t.Data = dataType
			t.SkelName = dataType.SkelName
			for _, typeArg := range t.TypeArguments {
				fixTypeRef(typeArg, refCtx)
			}
			checkTypeArguments(t, refName)
			return
		}
		t.Kind = model.TypeKindTypeParameter
		t.TypeParameter = param
		return

	case model.TypeKindList:
		fixTypeRef(t.List.Value, refCtx)

	case model.TypeKindMap:
		fixTypeRef(t.Map.Key, refCtx)
		checkTypeCanBeMapKey(t.Map.Key)
		fixTypeRef(t.Map.Value, refCtx)
	}
}

func checkTypeArguments(t *model.Type, refName string) {
	referencePos := t.ReferencePos
	if referencePos == (model.Position{}) {
		referencePos = t.Pos
	}
	if len(t.Data.TypeParameters) == 0 {
		checkutil.Check(len(t.TypeArguments) == 0,
			"%s data %s do not support type argument(s)", referencePos, refName)
		return
	}
	checkutil.Check(len(t.TypeArguments) > 0,
		"%s generic data %s need type argument(s)", referencePos, refName)
	checkutil.Check(len(t.Data.TypeParameters) == len(t.TypeArguments),
		"%s generic data %s have mismatched type arguments(s), found=%d, expected=%d",
		referencePos, refName, len(t.TypeArguments), len(t.Data.TypeParameters))
}
