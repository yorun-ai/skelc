package source

import (
	"path/filepath"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang/view"
)

func TestBuildDocGoPayloadUsesDomainDescription(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelWithDescriptionForTest("demo.user", "User domain"))
	gen := newGen(Option{
		Domain:      pkg,
		View:        view.New(view.ModeFull, pkg),
		Mode:        view.ModeFull,
		PackageName: "skeled",
		Out:         filepath.Join(t.TempDir(), "skeled"),
	})

	payload := gen.buildDocGoPayload()

	if len(payload.CommentLines) == 0 || payload.CommentLines[0] != "Package skeled User domain" {
		t.Fatalf("unexpected doc comment lines: %+v", payload.CommentLines)
	}
}

func TestBuildDocGoPayloadFallsBackToPackageName(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("demo.user"))
	gen := newGen(Option{
		Domain:      pkg,
		View:        view.New(view.ModeFull, pkg),
		Mode:        view.ModeFull,
		PackageName: "skeled",
		Out:         filepath.Join(t.TempDir(), "skeled"),
	})

	payload := gen.buildDocGoPayload()

	if len(payload.CommentLines) == 0 || payload.CommentLines[0] != "Package skeled" {
		t.Fatalf("unexpected fallback doc comment lines: %+v", payload.CommentLines)
	}
}
