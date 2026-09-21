package common

import (
	"go.yorun.ai/skelc/internal/model"
	"reflect"
	"strings"
	"testing"
)

func TestResolveImportAliases(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		types := []*model.Type{
			{ExternalDomain: "first.user", ExternalAlias: "userapi"},
			{ExternalDomain: "second.user", ExternalAlias: "userapi"},
			{ExternalDomain: "explicit", ExternalAlias: "firstuser", ExternalAliasExplicit: true},
			{ExternalDomain: "plain.order", ExternalAlias: "orderapi"},
			{ExternalDomain: "seconduser", ExternalAlias: "context", ExternalAliasExplicit: true},
		}
		if reversed {
			for i, j := 0, len(types)-1; i < j; i, j = i+1, j-1 {
				types[i], types[j] = types[j], types[i]
			}
		}
		ResolveImportAliases(types, func(domain string) string { return strings.ReplaceAll(domain, ".", "") }, []string{"context"})
		got := map[string]string{}
		for _, kind := range types {
			got[kind.ExternalDomain] = kind.ExternalAlias
		}
		want := map[string]string{"first.user": "firstuser2", "second.user": "seconduser2", "explicit": "firstuser", "plain.order": "orderapi", "seconduser": "seconduser"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("reversed=%v got %v want %v", reversed, got, want)
		}
	}
}
