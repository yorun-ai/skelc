package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidatePackageName(t *testing.T) {
	for _, name := range []string{"example", "example-name_2.js", "@example/package", "@example/sub_package"} {
		if err := ValidatePackageName(name); err != nil {
			t.Fatalf("rejected %q: %v", name, err)
		}
	}
	for _, name := range []string{"", "@scope", "@/package", "@scope/", "@scope/a/b", "@scope/.hidden", "name with space", "bad\"name", "bad\\name", "Uppercase", "_hidden", ".hidden", "-hidden", "node_modules", "favicon.ico", strings.Repeat("x", 215)} {
		t.Run(name, func(t *testing.T) {
			if err := ValidatePackageName(name); err == nil {
				t.Fatalf("accepted %q", name)
			}
			out := t.TempDir()
			if err := Generate(Option{Out: out, PackageName: name}); err == nil {
				t.Fatal("generated metadata for invalid package name")
			}
			if _, err := os.Stat(filepath.Join(out, "package.json")); !os.IsNotExist(err) {
				t.Fatalf("wrote invalid metadata: %v", err)
			}
		})
	}
}

func TestDependencyNamesAreValidated(t *testing.T) {
	for _, name := range []string{"bad\"name", "@scope", "@scope/a/b"} {
		if _, err := packageJSONDependencies(Option{Imports: map[string]string{"sample": name + "@1.0.0"}}); err == nil {
			t.Fatalf("accepted invalid dependency name %q", name)
		}
	}
}
