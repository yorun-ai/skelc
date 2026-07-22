package module

import "testing"

func TestImportPathStripsVersion(t *testing.T) {
	path := "go.yorun.ai/app/vine/demo/skeled/userpub@v0.0.0-00010101000000-000000000000"
	got, err := ImportPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "go.yorun.ai/app/vine/demo/skeled/userpub" {
		t.Fatalf("unexpected go import path: %s", got)
	}
}

func TestParseGoImportDependencyRejectsMissingVersion(t *testing.T) {
	if _, err := parseGoImportDependency("go.yorun.ai/app/userpub@"); err == nil {
		t.Fatal("expected missing version error")
	}
}
