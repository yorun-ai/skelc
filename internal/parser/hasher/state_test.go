package hasher

import "testing"

func TestFillHashesIncludesActorCredential(t *testing.T) {
	oldDomain := newHashActorCredentialTestDomain("subject")
	newDomain := newHashActorCredentialTestDomain("tenant")

	FillHashes(oldDomain)
	FillHashes(newDomain)

	if oldDomain.Actors()[0].Hash == newDomain.Actors()[0].Hash {
		t.Fatal("expected actor hash to change when credential changes")
	}
	if oldDomain.Hash() == newDomain.Hash() {
		t.Fatal("expected domain hash to change when actor credential changes")
	}
}
