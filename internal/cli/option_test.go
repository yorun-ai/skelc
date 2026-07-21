package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/parser"
)

func TestValidateGoGenOption(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{Out: "./gen/go"}

	validateGoGenOption(input, option)
}

func TestValidateGoGenOptionRequiresModuleName(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{Out: "./gen/go", AsModule: true}

	expectPanicContains(t, "missing flag go-module or go-module-prefix", func() {
		validateGoGenOption(input, option)
	})
}

func TestValidateGoGenOptionAllowsModulePrefix(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:          "./gen/go",
		AsModule:     true,
		ModulePrefix: "github.com/acme/skel",
	}

	validateGoGenOption(input, option)
}

func TestValidateGoGenOptionRejectsModuleForNonModuleOutput(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:    "./gen/go",
		Module: "github.com/acme/skel/demo/user",
	}

	expectPanicContains(t, "flag go-module requires go-module output", func() {
		validateGoGenOption(input, option)
	})
}

func TestValidateGoGenOptionRejectsPubModuleWithoutPubOut(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:          "./gen/go",
		AsModule:     true,
		ModulePrefix: "github.com/acme/skel",
		PubModule:    "github.com/acme/skel/demo/userpub",
	}

	expectPanicContains(t, "flag go-pub-module requires go-pub-out", func() {
		validateGoGenOption(input, option)
	})
}

func TestValidateGoGenOptionRejectsPubOutForNonModuleOutput(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.GolangOption{
		Out:          "./gen/go",
		PubOut:       "./gen/gopub",
		ModulePrefix: "github.com/acme/skel",
	}

	expectPanicContains(t, "flag go-pub-out requires go-module output", func() {
		validateGoGenOption(input, option)
	})
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
			expectPanicContains(t, test.expected, func() {
				validateGoGenOption(input, test.option)
			})
		})
	}
}

func TestValidateTypeScriptGenOptionRequiresModuleName(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.TypeScriptOption{Out: "./gen/ts", AsModule: true}

	expectPanicContains(t, "missing flag ts-module or ts-module-scope", func() {
		validateTypeScriptGenOption(input, option)
	})
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
			expectPanicContains(t, test.expected, func() {
				validateTypeScriptGenOption(input, test.option)
			})
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

	validateTypeScriptGenOption(input, option)
}

func TestValidateSkelGenOptionRequiresPub(t *testing.T) {
	input := skelc.Input{SkelIn: "./demo"}
	option := skelc.SkeletonOption{Out: "./gen/skel"}

	expectPanicContains(t, "flag pub is required for gen skel", func() {
		validateSkelGenOption(input, option)
	})
}

func TestNormalizeCheckOption(t *testing.T) {
	parserOption := parser.Option{SkelIn: "./demo"}

	normalizeCheckOption(&parserOption)
	if !filepath.IsAbs(parserOption.SkelIn) {
		t.Fatalf("expected absolute skel-in, got %q", parserOption.SkelIn)
	}
}

func TestNormalizeCheckOptionRequiresInput(t *testing.T) {
	parserOption := parser.Option{}
	expectPanicContains(t, "missing flag skel-in", func() {
		normalizeCheckOption(&parserOption)
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
