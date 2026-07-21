package analyzer

import (
	"testing"

	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

func TestParseTypeAndFixRef(t *testing.T) {
	page := &model.Data{
		Name: "Page",
		TypeParameters: []*model.TypeParameter{
			{Name: "TItem"},
		},
	}
	user := &model.Data{Name: "User"}

	tp := parseType(refGrammarType("Page", refGrammarType("User")))
	fixTypeRef(tp, &refContext{
		dataList: map[string]*model.Data{
			"Page": page,
			"User": user,
		},
		typeParameters: map[string]*model.TypeParameter{
			"TItem": page.TypeParameters[0],
		},
	})

	if tp.Kind != model.TypeKindData {
		t.Fatalf("unexpected type kind: %v", tp.Kind)
	}
	if tp.Data != page {
		t.Fatalf("unexpected data: %+v", tp.Data)
	}
	if len(tp.TypeArguments) != 1 || tp.TypeArguments[0].Data != user {
		t.Fatalf("unexpected type arguments: %+v", tp.TypeArguments)
	}
	if tp.Name() != "PageOfUser" {
		t.Fatalf("unexpected type name: %s", tp.Name())
	}
}

func TestTypeRefData(t *testing.T) {
	user := &model.Data{Name: "User"}
	page := &model.Type{
		Kind: model.TypeKindData,
		Data: &model.Data{Name: "Page"},
		TypeArguments: []*model.Type{
			{Kind: model.TypeKindData, Data: user},
		},
	}
	refs := referencedData(page)

	if refs[page.Data] != _rkDirect {
		t.Fatalf("unexpected direct ref kind: %v", refs[page.Data])
	}
	if refs[user] != _rkDirect {
		t.Fatalf("unexpected nested ref kind: %v", refs[user])
	}
}

func TestParseTypeMapAndNullable(t *testing.T) {
	typ := parseType(nullableType(mapType(plainType(grammar.String), refGrammarType("User"))))
	if typ.Kind != model.TypeKindMap {
		t.Fatalf("unexpected type kind: %v", typ.Kind)
	}
	if !typ.Nullable {
		t.Fatal("expected nullable map type")
	}
	if typ.Map.Key.Kind != model.TypeKindScalar || typ.Map.Key.Scalar != model.ScalarString {
		t.Fatalf("unexpected map key: %+v", typ.Map.Key)
	}
	if typ.Map.Value.Kind != typeKindReference || typ.Map.Value.SkelName != "User" {
		t.Fatalf("unexpected map value: %+v", typ.Map.Value)
	}
}

func TestFixRefReturnsErrorWhenDefinitionMissing(t *testing.T) {
	typ := parseType(refGrammarType("User"))

	expectPanicContains(t, "definition of User not found", func() {
		fixTypeRef(typ, &refContext{})
	})
}

func TestFixRefReturnsErrorWhenGenericTypeArgsMismatch(t *testing.T) {
	page := &model.Data{
		Name: "Page",
		TypeParameters: []*model.TypeParameter{
			{Name: "TItem"},
		},
	}

	typ := parseType(refGrammarType("Page", refGrammarType("User"), refGrammarType("Profile")))

	expectPanicContains(t, "mismatched type arguments", func() {
		fixTypeRef(typ, &refContext{
			dataList: map[string]*model.Data{
				"Page":    page,
				"User":    {Name: "User"},
				"Profile": {Name: "Profile"},
			},
		})
	})
}

func TestFixRefReturnsErrorWhenGenericTypeArgsMissing(t *testing.T) {
	page := &model.Data{
		Name: "Page",
		TypeParameters: []*model.TypeParameter{
			{Name: "TItem"},
		},
	}

	typ := parseType(refGrammarType("Page"))

	expectPanicContains(t, "need type argument", func() {
		fixTypeRef(typ, &refContext{
			dataList: map[string]*model.Data{
				"Page": page,
			},
		})
	})
}

func TestFixRefReturnsErrorWhenMapKeyIsNullable(t *testing.T) {
	typ := parseType(mapType(nullableType(plainType(grammar.String)), plainType(grammar.Int)))

	expectPanicContains(t, "incorrect key type, must not be nullable", func() {
		fixTypeRef(typ, &refContext{})
	})
}

func TestFixRefReturnsErrorWhenMapKeyTypeIsUnsupported(t *testing.T) {
	typ := parseType(mapType(plainType(grammar.Float), plainType(grammar.Int)))

	expectPanicContains(t, "int/string or Enum expected", func() {
		fixTypeRef(typ, &refContext{})
	})
}
