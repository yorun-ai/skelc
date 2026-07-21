package analyzer

import (
	"slices"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

type _RefKind int

const (
	_rkNone _RefKind = iota
	_rkDirect
	_rkNullable
	_rkList
	_rkMap
)

var hardRefKinds = []_RefKind{_rkDirect}

func (rk _RefKind) isHard() bool {
	return slices.Contains(hardRefKinds, rk)
}

type _Refs map[*model.Data]_RefKind

func (r _Refs) put(rk _RefKind, rs *model.Data) {
	if prevKind, exists := r[rs]; exists && prevKind <= rk {
		return
	}
	r[rs] = rk
}

func (r _Refs) merge(other _Refs) {
	for refData, refKind := range other {
		r.put(refKind, refData)
	}
}

func (r _Refs) override(refKind _RefKind, other _Refs) {
	for refData := range other {
		r.put(refKind, refData)
	}
}

type _RefsMatrix map[*model.Data]_Refs

func (rm _RefsMatrix) has(src *model.Data) bool {
	_, exists := rm[src]
	return exists
}

func (rm _RefsMatrix) refKind(src *model.Data, dst *model.Data) _RefKind {
	if _, exists := rm[src]; !exists {
		return _rkNone
	}
	if _, exists := rm[src][dst]; !exists {
		return _rkNone
	}
	return rm[src][dst]
}

func referencedData(t *model.Type) _Refs {
	refs := _Refs{}
	switch t.Kind {
	case model.TypeKindData:
		rk := _rkDirect
		if t.Nullable {
			rk = _rkNullable
		}
		refs.put(rk, t.Data)
		for _, arg := range t.TypeArguments {
			refs.override(rk, referencedData(arg))
		}
	case model.TypeKindList:
		refs.override(_rkList, referencedData(t.List.Value))
	case model.TypeKindMap:
		refs.override(_rkMap, referencedData(t.Map.Value))
	}
	return refs
}

func checkTypeCanBeMapKey(t *model.Type) {
	checkutil.CheckNot(t.Nullable, "%s incorrect key type, must not be nullable", t.Pos)

	canBeMapKey := false
	if slices.Contains(mapKeyTypes, t.Kind) {
		if t.Kind != model.TypeKindScalar {
			canBeMapKey = true
		} else if slices.Contains(mapKeyScalarTypes, t.Scalar) {
			canBeMapKey = true
		}
	}
	checkutil.Check(canBeMapKey, "%s incorrect key type, int/string or Enum expected", t.Pos)
}

func parseType(s *grammar.Type) *model.Type {
	if s == nil {
		return nil
	}

	t := &model.Type{
		Pos:           position(s.Pos),
		Kind:          typeKindNone,
		Scalar:        scalarNone,
		List:          nil,
		Map:           nil,
		Data:          nil,
		TypeParameter: nil,
		Nullable:      false,
	}

	switch {
	case s.Plain != nil:
		t.Kind = model.TypeKindScalar
		switch *s.Plain {
		case grammar.Int:
			t.Scalar = model.ScalarInt
		case grammar.Float:
			t.Scalar = model.ScalarFloat
		case grammar.Boolean:
			t.Scalar = model.ScalarBoolean
		case grammar.String:
			t.Scalar = model.ScalarString
		case grammar.Decimal:
			t.Scalar = model.ScalarDecimal
		case grammar.Binary:
			t.Scalar = model.ScalarBinary
		case grammar.Timestamp:
			t.Scalar = model.ScalarTimestamp
		case grammar.Duration:
			t.Scalar = model.ScalarDuration
		case grammar.LocalDate:
			t.Scalar = model.ScalarLocalDate
		case grammar.LocalTime:
			t.Scalar = model.ScalarLocalTime
		case grammar.LocalDateTime:
			t.Scalar = model.ScalarLocalDateTime
		case grammar.UUID:
			t.Scalar = model.ScalarUUID
		case grammar.JSON:
			t.Scalar = model.ScalarJSON
		case grammar.SkelPermissionCode:
			t.Kind = model.TypeKindSkelPermissionCode
		default:
			checkutil.Panicf("%s unknown PlainType %s", s.Pos, *s.Plain)
		}

	case s.List != nil:
		valueType := parseType(s.List.Value)
		t.Kind = model.TypeKindList
		t.List = &model.ListType{
			Value: valueType,
		}

	case s.Map != nil:
		keyType := parseType(s.Map.Key)
		valueType := parseType(s.Map.Value)
		t.Kind = model.TypeKindMap
		t.Map = &model.MapType{
			Key:   keyType,
			Value: valueType,
		}

	case s.Reference != nil:
		refName, refQualifier, refPos := parseReferenceName(s.Reference.Name)
		checkCase("Enum/Data/TypeParameter", caseTypeCamel, &grammar.Identifier{Value: refName, Pos: refPos})
		t.Kind = typeKindReference
		typeArgs := make([]*model.Type, 0, len(s.Reference.TypeArguments))
		for _, typeArg := range s.Reference.TypeArguments {
			parsedTypeArg := parseType(typeArg)
			typeArgs = append(typeArgs, parsedTypeArg)
		}
		t.ReferencePos = position(refPos)
		t.SkelName = refName
		t.ExternalAlias = refQualifier
		t.TypeArguments = typeArgs

	default:
		checkutil.Panicf("%s unknown Type %+v", s.Pos, s)
	}

	t.Nullable = s.Nullable

	return t
}

func parseReferenceName(name *grammar.QualifiedName) (string, string, lexer.Position) {
	checkutil.Check(name != nil && len(name.Parts) > 0, "missing reference type")
	if len(name.Parts) == 1 {
		return name.Parts[0].Value, "", name.Parts[0].Pos
	}
	if len(name.Parts) == 2 {
		return name.Parts[1].Value, name.Parts[0].Value, name.Parts[1].Pos
	}
	checkutil.Check(len(name.Parts) <= 2, "%s reference type supports at most one import qualifier", name.Pos)
	return name.Parts[1].Value, name.Parts[0].Value, name.Parts[1].Pos
}
