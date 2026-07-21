package module

import (
	"fmt"
	"strings"
	"testing"
)

func TestImportPathStripsVersion(t *testing.T) {
	path := "go.yorun.ai/app/vine/demo/skeled/userpub@v0.0.0-00010101000000-000000000000"
	if got := ImportPath(path); got != "go.yorun.ai/app/vine/demo/skeled/userpub" {
		t.Fatalf("unexpected go import path: %s", got)
	}
}

func TestParseGoImportDependencyRejectsMissingVersion(t *testing.T) {
	expectPanicContains(t, `invalid go import "go.yorun.ai/app/userpub@": missing version`, func() {
		parseGoImportDependency("go.yorun.ai/app/userpub@")
	})
}

func expectPanicContains(t *testing.T, expected string, fn func()) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic containing %q", expected)
		}
		if !strings.Contains(fmt.Sprint(recovered), expected) {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()
	fn()
}
