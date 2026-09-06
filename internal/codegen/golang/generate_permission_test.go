package golang_test

import (
	"go.yorun.ai/skelc"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPermissionGenerationUsesStrings(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "domain.skel")
	contract := `domain demo
pub data Permissions { code: string
 optional: string? }
pub resource User {
    check byId { input {
        id: string
    } }
    action read
    action update
}
actor ClientActor { via client {} }
service UserService {
    for ClientActor
    method update {
        require User:update:byId(id)
        input { id: string }
    }
}`
	if err := os.WriteFile(input, []byte(contract), 0600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "generated")
	if _, err := skelc.CompileGolang(skelc.Input{SkelIn: input}, skelc.GolangOption{Out: out, AsModule: true, Module: "example.com/permissions"}); err != nil {
		t.Fatal(err)
	}
	// TODO: Remove these legacy-output assertions after the PermissionCode migration period ends.
	for _, name := range []string{"data.go", "resource.go", "schema.go"} {
		content := readFileForTest(t, filepath.Join(out, name))
		for _, forbidden := range []string{"skel.PermissionCode", "skel.TypeKindSkelPermissionCode", "UserPermissionCodes"} {
			if strings.Contains(content, forbidden) {
				t.Fatalf("%s contains %s", name, forbidden)
			}
		}
	}
	resource := strings.Join(strings.Fields(readFileForTest(t, filepath.Join(out, "resource.go"))), " ")
	if !strings.Contains(resource, `UserReadPermission string = "demo.User:read"`) || !strings.Contains(resource, "code string") {
		t.Fatalf("expected string constant and check parameter: %s", resource)
	}
	tsOut := filepath.Join(root, "ts")
	if _, err := skelc.CompileTypeScript(skelc.Input{SkelIn: input}, skelc.TypeScriptOption{Out: tsOut}); err != nil {
		t.Fatal(err)
	}
	ts := strings.Join(strings.Fields(readFileForTest(t, filepath.Join(tsOut, "data.ts"))), " ")
	if !strings.Contains(ts, "code: string") || !strings.Contains(ts, "optional: string | null") {
		t.Fatalf("expected string TypeScript fields: %s", ts)
	}
	public := filepath.Join(root, "public")
	if _, err := skelc.CompileSkeleton(skelc.Input{SkelIn: input}, skelc.SkeletonOption{Out: public, PubOnly: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := skelc.CompileGolang(skelc.Input{SkelIn: public}, skelc.GolangOption{Out: filepath.Join(root, "roundtrip")}); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-mod=mod", "./...")
	command.Dir = out
	command.Env = append(os.Environ(), "GOWORK=off")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated module: %v\n%s", err, output)
	}
}
