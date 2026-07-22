package hasher

import (
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestFillHashesPropagatesDataChangesToService(t *testing.T) {
	oldDomain := newHashTestDomain(t, "User service")
	newDomain := newHashTestDomain(t, "User service")
	newDomain.Data()[0].Members = append(newDomain.Data()[0].Members, &model.DataMember{
		Name: "nickname",
		Type: &model.Type{
			Kind:   model.TypeKindScalar,
			Scalar: model.ScalarString,
		},
	})

	FillHashes(oldDomain)
	FillHashes(newDomain)

	if oldDomain.Data()[0].Hash == newDomain.Data()[0].Hash {
		t.Fatal("expected data hash to change")
	}
	if oldDomain.Services()[0].Methods[0].Hash == newDomain.Services()[0].Methods[0].Hash {
		t.Fatal("expected method hash to change")
	}
	if oldDomain.Services()[0].Hash == newDomain.Services()[0].Hash {
		t.Fatal("expected service hash to change")
	}
	if oldDomain.Hash() == newDomain.Hash() {
		t.Fatal("expected domain hash to change")
	}
}

func TestFillHashesIncludesAllowVia(t *testing.T) {
	clientDomain := newHashAllowViaTestDomain(t, "client")
	openapiDomain := newHashAllowViaTestDomain(t, "openapi")

	FillHashes(clientDomain)
	FillHashes(openapiDomain)

	if clientDomain.Services()[0].Hash == openapiDomain.Services()[0].Hash {
		t.Fatal("expected service hash to change when for via changes")
	}
	if clientDomain.Webs()[0].Hash == openapiDomain.Webs()[0].Hash {
		t.Fatal("expected web hash to change when for via changes")
	}
	if clientDomain.Hash() == openapiDomain.Hash() {
		t.Fatal("expected domain hash to change when for via changes")
	}
}
