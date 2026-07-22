package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/parser"
)

func TestValidateGoGenOption(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{Out: "./gen/go"}

	if err := validateGoGenOption(input, option); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGoGenOptionRequiresModuleName(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{Out: "./gen/go", AsModule: true}

	expectOptionError(t, validateGoGenOption(input, option), "missing flag go-module or go-module-prefix")
}

func TestValidateGoGenOptionAllowsModulePrefix(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:          "./gen/go",
		AsModule:     true,
		ModulePrefix: "github.com/acme/skel",
	}

	if err := validateGoGenOption(input, option); err != nil {
		t.Fatal(err)
	}
}

func TestValidateGoGenOptionRejectsModuleForNonModuleOutput(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:    "./gen/go",
		Module: "github.com/acme/skel/demo/user",
	}

	expectOptionError(t, validateGoGenOption(input, option), "flag go-module requires go-module output")
}

func TestValidateGoGenOptionRejectsPubModuleWithoutPubOut(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:          "./gen/go",
		AsModule:     true,
		ModulePrefix: "github.com/acme/skel",
		PubModule:    "github.com/acme/skel/demo/userpub",
	}

	expectOptionError(t, validateGoGenOption(input, option), "flag go-pub-module requires go-pub-out")
}

func TestValidateGoGenOptionRejectsPubOutForNonModuleOutput(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:          "./gen/go",
		PubOut:       "./gen/gopub",
		ModulePrefix: "github.com/acme/skel",
	}

	expectOptionError(t, validateGoGenOption(input, option), "flag go-pub-out requires go-module output")
}

func TestValidateGoGenOptionRejectsTrailingSlash(t *testing.T) {
	tests := []struct {
		name     string
		option   skelc.GolangOption
		expected string
	}{
		{
			name: "module prefix",
			option: skelc.GolangOption{
				Out:          "./gen/go",
				AsModule:     true,
				ModulePrefix: "github.com/acme/skel/",
			},
			expected: "flag go-module-prefix must not end with /",
		},
		{
			name: "module",
			option: skelc.GolangOption{
				Out:      "./gen/go",
				AsModule: true,
				Module:   "github.com/acme/skel/demo/user/",
			},
			expected: "flag go-module must not end with /",
		},
		{
			name: "import",
			option: skelc.GolangOption{
				Out:          "./gen/go",
				AsModule:     true,
				ModulePrefix: "github.com/acme/skel",
				Imports:      map[string]string{"user": "github.com/acme/skel/demo/userpub/"},
			},
			expected: "flag go-import must not end with /",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := skelc.Input{SkelIn: "./demo"}
			expectOptionError(t, validateGoGenOption(input, test.option), test.expected)
		})
	}
}

func TestValidateTypeScriptGenOptionRequiresModuleName(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.TypeScriptOption{Out: "./gen/ts", AsModule: true}

	expectOptionError(t, validateTypeScriptGenOption(input, option), "missing flag ts-module or ts-module-scope")
}

func TestValidateTypeScriptGenOptionRejectsModuleOptionsWithoutAsModule(t *testing.T) {
	tests := []struct {
		name     string
		option   skelc.TypeScriptOption
		expected string
	}{
		{
			name:     "module",
			option:   skelc.TypeScriptOption{Out: "./gen/ts", Module: "@acme/skeled-user"},
			expected: "flag ts-module requires ts-as-module",
		},
		{
			name:     "module scope",
			option:   skelc.TypeScriptOption{Out: "./gen/ts", ModuleScope: "@acme/skeled"},
			expected: "flag ts-module-scope requires ts-as-module",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := skelc.Input{SkelIn: "./demo"}
			expectOptionError(t, validateTypeScriptGenOption(input, test.option), test.expected)
		})
	}
}

func TestValidateTypeScriptGenOption(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.TypeScriptOption{
		Out:         "./gen/ts",
		AsModule:    true,
		ModuleScope: "@acme/skeled/",
		Module:      "@acme/skeled-user/",
		Imports: map[string]string{
			"user": "@acme/skeled-userpub/",
		},
	}

	if err := validateTypeScriptGenOption(input, option); err != nil {
		t.Fatal(err)
	}
}

func TestValidateSkelGenOptionRequiresPub(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.SkeletonOption{Out: "./gen/skel"}

	expectOptionError(t, validateSkelGenOption(input, option), "flag pub is required for gen skel")
}

func TestNormalizeCheckOption(t *testing.T) {
	parserOption := parser.Option{SkelIn: "./demo"}

	if err := normalizeCheckOption(&parserOption); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(parserOption.SkelIn) {
		t.Fatalf("expected absolute skel-in, got %q", parserOption.SkelIn)
	}
}

func TestNormalizeCheckOptionRequiresInput(t *testing.T) {
	parserOption := parser.Option{}
	expectOptionError(t, normalizeCheckOption(&parserOption), "missing flag skel-in")
}

func expectOptionError(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got %v", expected, err)
	}
}
