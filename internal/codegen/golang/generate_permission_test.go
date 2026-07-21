package golang_test

import (
	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratorRendersGoSkelChecks(t *testing.T) {
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
	if !strings.Contains(string(goDataContent), `import "go.yorun.ai/vine/core/rpc"`) {
		t.Fatalf("expected data.go rpc import, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), "func (v *Profile) Validate(path string) error") {
		t.Fatalf("expected Profile Validate, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `rpc.CheckValueNotNil(v.Aliases, rpc.JoinPath(path, "Aliases"))`) {
		t.Fatalf("expected aliases not nil check, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `(&v.Profile).Validate(rpc.JoinPath(path, "Profile"))`) {
		t.Fatalf("expected nested profile check, got:\n%s", string(goDataContent))
	}
	if strings.Contains(string(goDataContent), `rpc.CheckValueNotNil(v.Labels`) {
		t.Fatalf("did not expect nullable labels map check, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `rpc.CheckValueNotNil(v.Friends, rpc.JoinPath(path, "Friends"))`) {
		t.Fatalf("expected required nullable profile list check, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `if v.Friends[i0] != nil {`) {
		t.Fatalf("expected nullable list item nil guard, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `if err := v.Friends[i0].Validate(rpc.JoinIndex(rpc.JoinPath(path, "Friends"), i0)); err != nil {`) {
		t.Fatalf("expected nullable list item validation, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `rpc.CheckValueNotNil(v.ProfilesByName, rpc.JoinPath(path, "ProfilesByName"))`) {
		t.Fatalf("expected required nullable profile map check, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `if item0 != nil {`) {
		t.Fatalf("expected nullable map value nil guard, got:\n%s", string(goDataContent))
	}
	if !strings.Contains(string(goDataContent), `if err := item0.Validate(rpc.JoinMapKey(rpc.JoinPath(path, "ProfilesByName"), key0)); err != nil {`) {
		t.Fatalf("expected nullable map value validation, got:\n%s", string(goDataContent))
	}

	goServiceContent, err := os.ReadFile(filepath.Join(goOutDir, "service.go"))
	if err != nil {
		t.Fatalf("read go service file: %v", err)
	}
	if !strings.Contains(string(goServiceContent), "ValidateResult: func(value any) error {") {
		t.Fatalf("expected service result checker, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), "ValidateArguments: func(value any) error {") {
		t.Fatalf("expected service arguments checker, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `args := value.(*_UserServiceListUsersArguments)`) {
		t.Fatalf("expected typed arguments assertion, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `rpc.CheckValueNotNil(args.Friends, rpc.JoinPath("arguments", "Friends"))`) {
		t.Fatalf("expected required nullable argument list check, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `if args.Friends[i0] != nil {`) {
		t.Fatalf("expected nullable argument list item guard, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `if err := args.Friends[i0].Validate(rpc.JoinIndex(rpc.JoinPath("arguments", "Friends"), i0)); err != nil {`) {
		t.Fatalf("expected nullable argument list item validation, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `rpc.CheckValueNotNil(args.ProfilesByName, rpc.JoinPath("arguments", "ProfilesByName"))`) {
		t.Fatalf("expected required nullable argument map check, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `if item0 != nil {`) {
		t.Fatalf("expected nullable argument map value guard, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `if err := item0.Validate(rpc.JoinMapKey(rpc.JoinPath("arguments", "ProfilesByName"), key0)); err != nil {`) {
		t.Fatalf("expected nullable argument map value validation, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `rpc.CheckValueNotNil(ret, "result")`) {
		t.Fatalf("expected top-level result list not nil check, got:\n%s", string(goServiceContent))
	}
	if !strings.Contains(string(goServiceContent), `(&ret[i0]).Validate(rpc.JoinIndex("result", i0))`) {
		t.Fatalf("expected result item check, got:\n%s", string(goServiceContent))
	}
}
