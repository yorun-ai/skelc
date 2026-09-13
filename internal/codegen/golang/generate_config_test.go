package golang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/compiler"
)

func TestGeneratedConfigNoTrimTags(t *testing.T) {
	for _, lifecycle := range []string{"eternal", "instant"} {
		t.Run(lifecycle, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "config.skel")
			source := "domain demo\nconfig TextConfig " + lifecycle + ` {
    plain: string
    @noTrim
    raw: string
    @noTrim
    optional: string?
    @noTrim
    items: list<string?>?
    @sensitive
    @noTrim
    values: map<string, string?>?
    @sensitive
    secret: string
}`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			parsed, err := compiler.Compile(compiler.Option{SkelIn: input})
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(dir, "generated")
			if err := golang.Generate(parsed.Domain, golang.Option{Out: out}); err != nil {
				t.Fatal(err)
			}
			content := readFileForTest(t, filepath.Join(out, "config.go"))
			for _, tag := range []string{
				"`json:\"plain\" yaml:\"plain\"`", `json:"raw" yaml:"raw" skel:"noTrim"`, `json:"optional" yaml:"optional" skel:"noTrim"`,
				`json:"items" yaml:"items" skel:"noTrim"`, `json:"values" yaml:"values" skel:"sensitive,noTrim"`, `json:"secret" yaml:"secret" skel:"sensitive"`,
			} {
				if !strings.Contains(content, tag) {
					t.Fatalf("missing %s in:\n%s", tag, content)
				}
			}
		})
	}
}

func TestGeneratedConfigMapEnumValues(t *testing.T) {
	for _, lifecycle := range []string{"eternal", "instant"} {
		t.Run(lifecycle, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "config.skel")
			source := "domain demo\nenum Mode { ACTIVE DISABLED }\nconfig AppConfig " + lifecycle + ` {
    modes: map<string, Mode>
    optionalModes: map<string, Mode?>?
    transitions: map<Mode, Mode>
}`
			if err := os.WriteFile(input, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			parsed, err := compiler.Compile(compiler.Option{SkelIn: input})
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(dir, "generated")
			if err := golang.Generate(parsed.Domain, golang.Option{Out: out}); err != nil {
				t.Fatal(err)
			}
			content := strings.Join(strings.Fields(readFileForTest(t, filepath.Join(out, "config.go"))), " ")
			for _, field := range []string{
				"Modes map[string]Mode",
				"OptionalModes *map[string]*Mode",
				"Transitions map[Mode]Mode",
			} {
				if !strings.Contains(content, field) {
					t.Fatalf("missing %s in:\n%s", field, content)
				}
			}
		})
	}
}
