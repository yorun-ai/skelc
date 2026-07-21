package golang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/model"
)

func TestGeneratorSkipsGoModuleFilesByDefault(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "skeled")

	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{
			{
				Name: "User",
				Members: []*model.DataMember{
					{Name: "id", Type: stringTypeForTest()},
				},
			},
		},
	})

	golang.Generate(pkg, golang.Option{Out: goOutDir})

	assertFileMissing(t, filepath.Join(goOutDir, "go.mod"))
	assertFileMissing(t, filepath.Join(goOutDir, "go.sum"))

	if _, err := os.ReadFile(filepath.Join(goOutDir, "doc.go")); err != nil {
		t.Fatalf("expected doc.go to exist: %v", err)
	}
	if _, err := os.ReadFile(filepath.Join(goOutDir, "data.go")); err != nil {
		t.Fatalf("expected data.go to exist: %v", err)
	}
}

func TestGeneratorRendersGoModuleFiles(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "golang")

	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{
			{
				Name: "User",
				Members: []*model.DataMember{
					{Name: "id", Type: stringTypeForTest()},
				},
			},
		},
	})

	golang.Generate(pkg, golang.Option{
		Out:          goOutDir,
		AsModule:     true,
		ModulePrefix: "github.com/acme/skel",
		VineVersion:  "v9.8.7",
	})

	if _, err := os.ReadFile(filepath.Join(goOutDir, "go.mod")); err != nil {
		t.Fatalf("expected go.mod to exist: %v", err)
	}
	assertFileMissing(t, filepath.Join(goOutDir, "go.sum"))
	goModContent, err := os.ReadFile(filepath.Join(goOutDir, "go.mod"))
	if err != nil {
		t.Fatalf("expected go.mod to exist: %v", err)
	}
	if !strings.Contains(string(goModContent), "module github.com/acme/skel/demo/user") {
		t.Fatalf("unexpected go.mod content: %q", string(goModContent))
	}
	if !strings.Contains(string(goModContent), "go.yorun.ai/vine v9.8.7") {
		t.Fatalf("unexpected go.mod content: %q", string(goModContent))
	}
}

func TestGeneratorRendersDefaultGoPubModulePrefix(t *testing.T) {
	goOutDir := filepath.Join(t.TempDir(), "golang")
	goPubOutDir := filepath.Join(t.TempDir(), "golangpub")

	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "user",
		Data: []*model.Data{
			{
				Pub:  true,
				Name: "User",
				Members: []*model.DataMember{
					{Name: "id", Type: stringTypeForTest()},
				},
			},
		},
	})

	golang.Generate(pkg, golang.Option{
		Out:          goOutDir,
		AsModule:     true,
		PubOut:       goPubOutDir,
		ModulePrefix: "go.yorun.ai/app/vine/demo",
		VineVersion:  "v9.8.7",
	})

	goModContent, err := os.ReadFile(filepath.Join(goPubOutDir, "go.mod"))
	if err != nil {
		t.Fatalf("expected go.mod to exist: %v", err)
	}
	if !strings.Contains(string(goModContent), "module go.yorun.ai/app/vine/demo/userpub") {
		t.Fatalf("unexpected go.mod content: %q", string(goModContent))
	}
}

func TestGeneratorRendersGoPubAndRegularModules(t *testing.T) {
	tmpDir := t.TempDir()
	goOutDir := filepath.Join(tmpDir, "user")
	goPubOutDir := filepath.Join(tmpDir, "userpub")

	user := &model.Data{
		Pub:  true,
		Name: "User",
		Members: []*model.DataMember{
			{Name: "id", Type: stringTypeForTest()},
		},
	}
	pkg := newModelDomainForTest(t, model.DomainSpec{
		Name: "demo.user",
		Data: []*model.Data{user},
		Services: []*model.Service{
			{
				Pub:  true,
				Name: "UserService",
				Methods: []*model.Method{
					methodForTest("UserService", &model.Method{Name: "getUser", ResultType: dataTypeForTest(user)}),
				},
			},
		},
		Events: []*model.Data{
			{
				Pub:  true,
				Name: "UserChangedEvent",
				Members: []*model.DataMember{
					{Name: "user", Type: dataTypeForTest(user)},
				},
			},
		},
	})

	golang.Generate(pkg, golang.Option{
		Out:          goOutDir,
		AsModule:     true,
		PubOut:       goPubOutDir,
		ModulePrefix: "github.com/acme/skel",
		VineVersion:  "v9.8.7",
	})

	pubServiceContent := readFileForTest(t, filepath.Join(goPubOutDir, "service.go"))
	if !strings.Contains(pubServiceContent, "rpc.ServiceSpecTypeClient") {
		t.Fatalf("expected pub service client spec, got:\n%s", pubServiceContent)
	}
	if strings.Contains(pubServiceContent, "type UserServiceServer interface") {
		t.Fatalf("did not expect pub service server, got:\n%s", pubServiceContent)
	}
	pubEventContent := readFileForTest(t, filepath.Join(goPubOutDir, "event.go"))
	if !strings.Contains(pubEventContent, "event.EventSpecTypeListener") {
		t.Fatalf("expected pub event listener spec, got:\n%s", pubEventContent)
	}
	if strings.Contains(pubEventContent, "type UserChangedEventEmitter interface") {
		t.Fatalf("did not expect pub event emitter, got:\n%s", pubEventContent)
	}

	regularServiceContent := readFileForTest(t, filepath.Join(goOutDir, "service.go"))
	if !strings.Contains(regularServiceContent, "rpc.ServiceSpecTypeServer") {
		t.Fatalf("expected regular pub service server spec, got:\n%s", regularServiceContent)
	}
	if strings.Contains(regularServiceContent, "type UserServiceClient interface") {
		t.Fatalf("did not expect regular pub service client, got:\n%s", regularServiceContent)
	}
	regularEventContent := readFileForTest(t, filepath.Join(goOutDir, "event.go"))
	if !strings.Contains(regularEventContent, "event.EventSpecTypeEmitter") {
		t.Fatalf("expected regular pub event emitter spec, got:\n%s", regularEventContent)
	}
	if strings.Contains(regularEventContent, "type UserChangedEventListener interface") {
		t.Fatalf("did not expect regular pub event listener, got:\n%s", regularEventContent)
	}

	pubSchemaContent := readFileForTest(t, filepath.Join(goPubOutDir, "schema.go"))
	if !strings.Contains(pubSchemaContent, `Name:     "User"`) ||
		!strings.Contains(pubSchemaContent, `Name:     "UserService"`) ||
		!strings.Contains(pubSchemaContent, `Name:     "UserChangedEvent"`) {
		t.Fatalf("expected pub schemas in pub schema.go, got:\n%s", pubSchemaContent)
	}
	if !strings.Contains(pubSchemaContent, "Full:   false") {
		t.Fatalf("expected pub schema to render full=false, got:\n%s", pubSchemaContent)
	}
	regularSchemaContent := readFileForTest(t, filepath.Join(goOutDir, "schema.go"))
	if !strings.Contains(regularSchemaContent, "Full:   true") ||
		!strings.Contains(regularSchemaContent, `Name:     "User"`) ||
		!strings.Contains(regularSchemaContent, `Name:     "UserService"`) ||
		!strings.Contains(regularSchemaContent, `Name:     "UserChangedEvent"`) {
		t.Fatalf("expected full regular schema in regular schema.go, got:\n%s", regularSchemaContent)
	}

	regularGoModContent := readFileForTest(t, filepath.Join(goOutDir, "go.mod"))
	if !strings.Contains(regularGoModContent, "github.com/acme/skel/demo/userpub") {
		t.Fatalf("expected regular module to require pub module, got:\n%s", regularGoModContent)
	}
	facadeContent := readFileForTest(t, filepath.Join(goOutDir, "pub.go"))
	if !strings.Contains(facadeContent, `import "github.com/acme/skel/demo/userpub"`) ||
		!strings.Contains(facadeContent, "type User = userpub.User") ||
		!strings.Contains(facadeContent, "type UserServiceClient = userpub.UserServiceClient") ||
		!strings.Contains(facadeContent, "type UserChangedEventListener = userpub.UserChangedEventListener") {
		t.Fatalf("expected regular pub facade, got:\n%s", facadeContent)
	}
}
