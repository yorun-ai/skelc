package hasher

import "testing"

func TestFillHashesChangesOnlyAffectedComponents(t *testing.T) {
	oldDomain := newHashTestDomain("User service")
	newDomain := newHashTestDomain("User service (new version)")

	FillHashes(oldDomain)
	FillHashes(newDomain)

	if oldDomain.Actors()[0].Hash != newDomain.Actors()[0].Hash {
		t.Fatal("expected actor hash to remain stable")
	}
	if oldDomain.Data()[0].Hash != newDomain.Data()[0].Hash {
		t.Fatal("expected data hash to remain stable")
	}
	if oldDomain.Services()[0].Hash == newDomain.Services()[0].Hash {
		t.Fatal("expected service hash to change")
	}
	if oldDomain.Hash() == newDomain.Hash() {
		t.Fatal("expected domain hash to change")
	}
	if len(oldDomain.Hash()) != 8 {
		t.Fatalf("expected 8-char domain hash, got %q", oldDomain.Hash())
	}
}
