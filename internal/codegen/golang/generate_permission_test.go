package golang_test

import (
	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorOmitsLegacyGoSkelChecks(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "skeled")

	profile := &model.Data{
		Name: "Profile",
		Members: []*model.DataMember{
			{Name: "aliases", Type: listTypeForTest(stringTypeForTest())},
		},
	}
	user := &model.Data{
		Name: "User",
		Members: []*model.DataMember{
			{Name: "profile", Type: dataTypeForTest(profile)},
			{Name: "labels", Type: nullableTypeForTest(mapTypeForTest(stringTypeForTest(), stringTypeForTest()))},
			{Name: "friends", Type: listTypeForTest(nullableTypeForTest(dataTypeForTest(profile)))},
			{Name: "profilesByName", Type: mapTypeForTest(stringTypeForTest(), nullableTypeForTest(dataTypeForTest(profile)))},
		},
	}
	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{profile, user},
		Services: []*model.Service{
			{
				Name: "UserService",
				Methods: []*model.Method{
					methodForTest("UserService", &model.Method{
						Name:       "listUsers",
						ResultType: listTypeForTest(dataTypeForTest(user)),
						Arguments: []*model.Argument{
							{Name: "friends", Type: listTypeForTest(nullableTypeForTest(dataTypeForTest(profile)))},
							{
								Name: "profilesByName",
								Type: mapTypeForTest(stringTypeForTest(), nullableTypeForTest(dataTypeForTest(profile))),
							},
						},
					}),
				},
			},
		},
	})

	golang.Generate(pkg, golang.Option{Out: goOutDir})

	goDataContent, err := os.ReadFile(filepath.Join(goOutDir, "data.go"))
	if err != nil {
		t.Fatalf("read go data file: %v", err)
	}
	if strings.Contains(string(goDataContent), `"go.yorun.ai/vine/core/rpc"`) ||
		strings.Contains(string(goDataContent), " Validate(path string) error") ||
		strings.Contains(string(goDataContent), "CheckValueNotNil") {
		t.Fatalf("unexpected legacy data validation output:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), "Labels         *map[string]string") {
		t.Fatalf("expected nullable map pointer, got:\n%s", string(goDataContent))
	}

	goServiceContent, err := os.ReadFile(filepath.Join(goOutDir, "service.go"))
	if err != nil {
		t.Fatalf("read go service file: %v", err)
	}
	if !strings.Contains(string(goServiceContent), "ValidateArguments: nil,") ||
		!strings.Contains(string(goServiceContent), "ValidateResult: nil,") {
		t.Fatalf("expected nil service validation hooks, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), "CloneArguments: func(value any) any {") {
		t.Fatalf("expected service arguments clone, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), "CloneResult: func(value any) any {") {
		t.Fatalf("expected service result clone, got:\n%s", string(goServiceContent))
	}
	if strings.Contains(string(goServiceContent), "CheckValueNotNil") ||
		strings.Contains(string(goServiceContent), "JoinPath(") ||
		strings.Contains(string(goServiceContent), ".Validate(") {
		t.Fatalf("unexpected legacy service validation output:\n%s", string(goServiceContent))
	}
}

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
