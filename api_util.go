package skelc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/codegen/skeleton"
	"go.yorun.ai/skelc/internal/codegen/typescript"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

func normalizeInput(input Input) parser.Option {
	checkutil.Check(strings.TrimSpace(input.SkelIn) != "", "skel input is required")
	return parser.Option{
		SkelIn:      absolutePath(input.SkelIn),
		SkelImports: normalizePathMap(input.SkelImports),
	}
}

func normalizeGolangOption(option GolangOption) golang.Option {
	checkutil.Check(strings.TrimSpace(option.Out) != "", "Go output is required")
	if option.AsModule {
		checkutil.Check(option.Module != "" || option.ModulePrefix != "", "Go module or module prefix is required")
	} else {
		checkutil.Check(option.PubOut == "", "Go public output requires module generation")
		checkutil.Check(option.PubModule == "", "Go public module requires module generation")
		checkutil.Check(option.Module == "", "Go module requires module generation")
		checkutil.Check(option.ModulePrefix == "", "Go module prefix requires module generation")
	}
	checkutil.Check(option.PubModule == "" || option.PubOut != "", "Go public module requires public output")

	modulePrefix := strings.TrimSpace(option.ModulePrefix)
	module := strings.TrimSpace(option.Module)
	pubModule := strings.TrimSpace(option.PubModule)
	checkNoTrailingSlash(modulePrefix, "Go module prefix")
	checkNoTrailingSlash(module, "Go module")
	checkNoTrailingSlash(pubModule, "Go public module")

	return golang.Option{
		CompilerVersion: strings.TrimSpace(option.CompilerVersion),
		AsModule:        option.AsModule,
		Out:             absolutePath(option.Out),
		Module:          module,
		PubOut:          optionalAbsolutePath(option.PubOut),
		PubModule:       pubModule,
		Imports:         normalizeImportMap(option.Imports),
		ModulePrefix:    modulePrefix,
		VineVersion:     strings.TrimSpace(option.VineVersion),
	}
}

func normalizeTypeScriptOption(option TypeScriptOption) typescript.Option {
	checkutil.Check(strings.TrimSpace(option.Out) != "", "TypeScript output is required")
	moduleScope := strings.TrimRight(strings.TrimSpace(option.ModuleScope), "/")
	module := strings.TrimRight(strings.TrimSpace(option.Module), "/")
	if option.AsModule {
		checkutil.Check(module != "" || moduleScope != "", "TypeScript module or module scope is required")
	} else {
		checkutil.Check(module == "", "TypeScript module requires module generation")
		checkutil.Check(moduleScope == "", "TypeScript module scope requires module generation")
	}

	return typescript.Option{
		PubOnly:     option.PubOnly,
		AsModule:    option.AsModule,
		Out:         absolutePath(option.Out),
		Module:      module,
		Imports:     trimMapValues(option.Imports, "/"),
		ModuleScope: moduleScope,
	}
}

func normalizeSkeletonOption(option SkeletonOption) skeleton.Option {
	checkutil.Check(strings.TrimSpace(option.Out) != "", "Skel output is required")
	return skeleton.Option{PubOnly: option.PubOnly, Out: absolutePath(option.Out)}
}

func checkNoTrailingSlash(value string, field string) {
	checkutil.CheckNot(strings.HasSuffix(value, "/"), "%s must not end with /", field)
}

func prepareOutputDir(dir string, noClean bool) {
	if noClean {
		return
	}
	err := os.MkdirAll(dir, 0o755)
	checkutil.CheckNilError(err, "create output directory %s", dir)
	entries, err := os.ReadDir(dir)
	checkutil.CheckNilError(err, "read output directory %s", dir)
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		err = os.RemoveAll(path)
		checkutil.CheckNilError(err, "remove %s in output directory %s", entry.Name(), dir)
	}
}

func absolutePath(path string) string {
	absPath, err := filepath.Abs(strings.TrimSpace(path))
	checkutil.CheckNilError(err, "resolve path %s", path)
	return absPath
}

func optionalAbsolutePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return absolutePath(path)
}

func normalizePathMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		normalized[key] = absolutePath(value)
	}
	return normalized
}

func normalizeImportMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		value = strings.TrimSpace(value)
		checkNoTrailingSlash(value, "Go import")
		normalized[key] = value
	}
	return normalized
}

func trimMapValues(values map[string]string, cutset string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	normalized := make(map[string]string, len(values))
	for key, value := range values {
		normalized[key] = strings.TrimRight(strings.TrimSpace(value), cutset)
	}
	return normalized
}

func recoverAPIError(err *error) {
	recovered := recover()
	if recovered == nil {
		return
	}
	if recoveredErr, ok := recovered.(error); ok {
		*err = recoveredErr
		return
	}
	*err = fmt.Errorf("%v", recovered)
}
