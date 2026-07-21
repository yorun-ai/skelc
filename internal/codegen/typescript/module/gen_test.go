package module

import (
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen"
)

func TestBuildPackageJSONPayload(t *testing.T) {
	payload := buildPackageJSONPayload(Option{PackageName: "@yorun-ai/skeled-example"})
	if payload.PackageName != "@yorun-ai/skeled-example" {
		t.Fatalf("unexpected package name: %s", payload.PackageName)
	}
	if got := joinPackageJSONDependencies(payload.PeerDependencies); got != "@yorun-ai/vrpc@*" {
		t.Fatalf("unexpected peer dependencies: %#v", payload.PeerDependencies)
	}
	output := codegen.RenderTemplate(packageJSONTemplate, payload)
	if !strings.Contains(output, `"@yorun-ai/vrpc": "*"`) {
		t.Fatalf("expected rendered package.json to include @yorun-ai/vrpc, got:\n%s", output)
	}
	if !strings.Contains(output, `"peerDependencies": {`) || strings.Contains(output, `"dependencies": {`) {
		t.Fatalf("expected rendered package.json to put deps in peerDependencies, got:\n%s", output)
	}
}

func TestBuildPackageJSONPayloadIncludesConfiguredAndResolvedImports(t *testing.T) {
	payload := buildPackageJSONPayload(Option{
		PackageName: "@yorun-ai/skeled-example",
		Imports: map[string]string{
			"demo.user": "@vine-demo/skeled-user@workspace:*",
			"app":       "@vine-demo/skeled-app",
		},
		ResolvedImports: map[string]string{
			"demo.user": "@vine-demo/skeled-user",
			"inventory": "@vine-demo/skeled-inventory",
		},
	})

	want := "@vine-demo/skeled-app@*,@vine-demo/skeled-inventory@*,@vine-demo/skeled-user@workspace:*,@yorun-ai/vrpc@*"
	if got := joinPackageJSONDependencies(payload.PeerDependencies); got != want {
		t.Fatalf("unexpected peer dependencies: got=%s want=%s", got, want)
	}
}

func TestImportPathStripsVersion(t *testing.T) {
	if got := ImportPath("@vine-demo/skeled-user@workspace:*"); got != "@vine-demo/skeled-user" {
		t.Fatalf("unexpected import path: %s", got)
	}
}

func TestPackageJSONTemplateUsesPureTypeScriptEntry(t *testing.T) {
	output := codegen.RenderTemplate(packageJSONTemplate, &PackageJSONPayload{
		PackageName: "@yorun-ai/skeled-example",
		PeerDependencies: []PackageJSONDependency{
			{Package: "@vine-demo/skeled-user", Version: "workspace:*"},
			{Package: "@yorun-ai/vrpc", Version: "*"},
		},
	})
	for _, expected := range []string{
		`"name": "@yorun-ai/skeled-example"`, `"private": true`,
		`"types": "./index.ts"`, `"default": "./index.ts"`, `"peerDependencies": {`,
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected package.json to contain %q, got:\n%s", expected, output)
		}
	}
	for _, forbidden := range []string{"./dist/index.js", "./dist/index.d.ts", "typescript", `"dependencies": {`} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("expected package.json to omit %q, got:\n%s", forbidden, output)
		}
	}
}

func joinPackageJSONDependencies(dependencies []PackageJSONDependency) string {
	values := make([]string, 0, len(dependencies))
	for _, dependency := range dependencies {
		values = append(values, dependency.Package+"@"+dependency.Version)
	}
	return strings.Join(values, ",")
}
