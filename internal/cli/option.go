package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/parser"
)

func validateGoGenOption(input skelc.Input, option skelc.GolangOption) error {
	if err := validateGenInput(input); err != nil {
		return err
	}
	if option.AsModule && option.Module == "" && option.ModulePrefix == "" {
		return fmt.Errorf("missing flag go-module or go-module-prefix")
	}
	if !option.AsModule {
		for _, item := range []struct{ value, message string }{
			{option.PubOut, "flag go-pub-out requires go-module output"},
			{option.PubModule, "flag go-pub-module requires go-module output"},
			{option.Module, "flag go-module requires go-module output"},
			{option.ModulePrefix, "flag go-module-prefix requires go-module output"},
		} {
			if item.value != "" {
				return fmt.Errorf("%s", item.message)
			}
		}
	}
	if option.PubModule != "" && option.PubOut == "" {
		return fmt.Errorf("flag go-pub-module requires go-pub-out")
	}
	if option.Out == "" {
		return fmt.Errorf("missing output flag")
	}
	for _, item := range []struct{ value, name string }{
		{option.ModulePrefix, "go-module-prefix"}, {option.Module, "go-module"}, {option.PubModule, "go-pub-module"},
	} {
		if err := checkNoTrailingSlash(item.value, item.name); err != nil {
			return err
		}
	}
	for _, value := range option.Imports {
		if err := checkNoTrailingSlash(value, "go-import"); err != nil {
			return err
		}
	}
	return nil
}

func validateTypeScriptGenOption(input skelc.Input, option skelc.TypeScriptOption) error {
	if err := validateGenInput(input); err != nil {
		return err
	}
	if option.AsModule && option.Module == "" && option.ModuleScope == "" {
		return fmt.Errorf("missing flag ts-module or ts-module-scope")
	}
	if !option.AsModule && option.Module != "" {
		return fmt.Errorf("flag ts-module requires ts-as-module")
	}
	if !option.AsModule && option.ModuleScope != "" {
		return fmt.Errorf("flag ts-module-scope requires ts-as-module")
	}
	if option.Out == "" {
		return fmt.Errorf("missing output flag")
	}
	return nil
}

func validateSkelGenOption(input skelc.Input, option skelc.SkeletonOption) error {
	if err := validateGenInput(input); err != nil {
		return err
	}
	if option.Out != "" && !option.PubOnly {
		return fmt.Errorf("flag pub is required for gen skel")
	}
	if option.Out == "" {
		return fmt.Errorf("missing output flag")
	}
	return nil
}

func validateGenInput(input skelc.Input) error {
	if input.SkelIn == "" {
		return fmt.Errorf("missing flag skel-in")
	}
	return nil
}

func normalizeParserOption(option *parser.Option) error {
	if option.SkelIn == "" {
		return fmt.Errorf("missing flag skel-in")
	}
	path, err := filepath.Abs(option.SkelIn)
	if err != nil {
		return fmt.Errorf("resolve path %s: %w", option.SkelIn, err)
	}
	option.SkelIn = path
	return nil
}

func normalizeCheckOption(option *parser.Option) error  { return normalizeParserOption(option) }
func normalizeSymbolOption(option *parser.Option) error { return normalizeParserOption(option) }

func checkNoTrailingSlash(value, flagName string) error {
	if value != "" && strings.HasSuffix(value, "/") {
		return fmt.Errorf("flag %s must not end with /", flagName)
	}
	return nil
}

func normalizeRequiredPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, err)
	}
	return absPath, nil
}
