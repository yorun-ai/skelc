package golang_test

import (
	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/model"
	"os"
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
