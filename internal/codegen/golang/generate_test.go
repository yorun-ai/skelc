package golang

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/model"
)

func TestNewGenDerivesModuleAndPackageName(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("demo.user.profile"))

	gen := newGen(_GenOption{
		VineVersion: "v0.9.0",
		Mode:        view.ModeFull,
		Domain:      pkg,
		Out:         filepath.Join(t.TempDir(), "skeled"),
	})

	if gen.modName != "" {
		t.Fatalf("unexpected module name: %s", gen.modName)
	}
	if gen.pkgName != "skeled" {
		t.Fatalf("unexpected package name: %s", gen.pkgName)
	}
}

func TestNewGenKeepsDomainDerivedPackageNameForModuleOutput(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("demo.user.profile"))

	gen := newGen(_GenOption{
		ModulePrefix: "github.com/acme/skel",
		VineVersion:  "v0.9.0",
		Mode:         view.ModeFull,
		Domain:       pkg,
		Out:          filepath.Join(t.TempDir(), "skeled"),
		AsModule:     true,
	})

	if gen.modName != "github.com/acme/skel/demo/user/profile" {
		t.Fatalf("unexpected module name: %s", gen.modName)
	}
	if gen.pkgName != "profile" {
		t.Fatalf("unexpected module package name: %s", gen.pkgName)
	}
}

func TestNewGenDerivesPubModuleAndPackageName(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("demo.user.profile"))

	gen := newGen(_GenOption{
		ModulePrefix: "github.com/acme/skel",
		VineVersion:  "v0.9.0",
		Mode:         view.ModePub,
		Domain:       pkg,
		Out:          filepath.Join(t.TempDir(), "skeled"),
		AsModule:     true,
	})

	if gen.modName != "github.com/acme/skel/demo/user/profilepub" {
		t.Fatalf("unexpected module name: %s", gen.modName)
	}
	if gen.pkgName != "profilepub" {
		t.Fatalf("unexpected module package name: %s", gen.pkgName)
	}
}

func TestNewGenRejectsInvalidLocalPackageNameFromOutputDir(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("demo.user.profile"))

	expectPanicContains(t, `go output directory name "my-skel go" is not a valid package name`, func() {
		newGen(_GenOption{
			VineVersion: "v0.9.0",
			Mode:        view.ModeFull,
			Domain:      pkg,
			Out:         filepath.Join(t.TempDir(), "my-skel go"),
		})
	})
}

func TestNewGenRejectsKeywordLocalPackageNameFromOutputDir(t *testing.T) {
	pkg := buildModelDomainForTest(t, domainModelForTest("demo.user.profile"))

	expectPanicContains(t, `go output directory name "go" is not a valid package name`, func() {
		newGen(_GenOption{
			VineVersion: "v0.9.0",
			Mode:        view.ModeFull,
			Domain:      pkg,
			Out:         filepath.Join(t.TempDir(), "go"),
		})
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

func buildModelDomainForTest(t *testing.T, spec model.DomainSpec) *model.Domain {
	t.Helper()
	return model.NewDomainFromSpec(spec)
}

func domainModelForTest(name string) model.DomainSpec {
	return model.DomainSpec{Name: name}
}
