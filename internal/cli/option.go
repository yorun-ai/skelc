package cli

import (
	"path/filepath"
	"strings"

	"go.yorun.ai/skelc"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

func validateGoGenOption(input skelc.Input, option skelc.GolangOption) {
	validateGenInput(input)

	if option.AsModule {
		checkutil.Check(option.Module != "" || option.ModulePrefix != "",
			"missing flag go-module or go-module-prefix")
	} else {
		checkutil.Check(option.PubOut == "", "flag go-pub-out requires go-module output")
		checkutil.Check(option.PubModule == "", "flag go-pub-module requires go-module output")
		checkutil.Check(option.Module == "", "flag go-module requires go-module output")
		checkutil.Check(option.ModulePrefix == "", "flag go-module-prefix requires go-module output")
	}
	checkutil.Check(option.PubModule == "" || option.PubOut != "",
		"flag go-pub-module requires go-pub-out")
	checkutil.Check(option.Out != "", "missing output flag")

	checkNoTrailingSlash(option.ModulePrefix, "go-module-prefix")
	checkNoTrailingSlash(option.Module, "go-module")
	checkNoTrailingSlash(option.PubModule, "go-pub-module")
	checkMapValuesNoTrailingSlash(option.Imports, "go-import")
}

func validateTypeScriptGenOption(input skelc.Input, option skelc.TypeScriptOption) {
	validateGenInput(input)

	if option.AsModule {
		checkutil.Check(option.Module != "" || option.ModuleScope != "",
			"missing flag ts-module or ts-module-scope")
	} else {
		checkutil.Check(option.Module == "", "flag ts-module requires ts-as-module")
		checkutil.Check(option.ModuleScope == "", "flag ts-module-scope requires ts-as-module")
	}
	checkutil.Check(option.Out != "", "missing output flag")

}

func validateSkelGenOption(input skelc.Input, option skelc.SkeletonOption) {
	validateGenInput(input)

	checkutil.Check(option.Out == "" || option.PubOnly, "flag pub is required for gen skel")
	checkutil.Check(option.Out != "", "missing output flag")
}

func validateGenInput(input skelc.Input) {
	checkutil.Check(input.SkelIn != "", "missing flag skel-in")
}

func normalizeCheckOption(parserOption *parser.Option) {
	checkutil.Check(parserOption.SkelIn != "", "missing flag skel-in")

	parserOption.SkelIn = normalizeRequiredPath(parserOption.SkelIn)
}

func normalizeSymbolOption(parserOption *parser.Option) {
	checkutil.Check(parserOption.SkelIn != "", "missing flag skel-in")

	parserOption.SkelIn = normalizeRequiredPath(parserOption.SkelIn)
}

func checkNoTrailingSlash(value string, flagName string) {
	if value == "" {
		return
	}
	checkutil.CheckNot(strings.HasSuffix(value, "/"), "flag %s must not end with /", flagName)
}

func checkMapValuesNoTrailingSlash(values map[string]string, flagName string) {
	for _, value := range values {
		checkNoTrailingSlash(value, flagName)
	}
}

func normalizeRequiredPath(path string) string {
	absPath, err := filepath.Abs(path)
	checkutil.CheckNilError(err, "resolve path %s", path)
	return absPath
}
