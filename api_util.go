package skelc

import (
	"fmt"
	gotoken "go/token"
	"path/filepath"
	"slices"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/codegen/golang"
	gomodule "go.yorun.ai/skelc/internal/codegen/golang/module"
	"go.yorun.ai/skelc/internal/codegen/skeleton"
	"go.yorun.ai/skelc/internal/codegen/typescript"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/model"
)

type _OptionValidationError struct {
	field   string
	rule    string
	message string
}

func (err *_OptionValidationError) Error() string           { return err.message }
func (err *_OptionValidationError) ValidationField() string { return err.field }
func (err *_OptionValidationError) ValidationRule() string  { return err.rule }

func optionValidationError(field, rule, message string) error {
	return &_OptionValidationError{field: field, rule: rule, message: message}
}

func normalizeInput(input Input) (parser.Option, error) {
	if strings.TrimSpace(input.SkelIn) == "" {
		return parser.Option{}, optionValidationError("skel.input", "required", "skel input is required")
	}
	skelIn, err := absolutePath(input.SkelIn)
	if err != nil {
		return parser.Option{}, err
	}
	imports, err := normalizePathMap(input.SkelImports)
	if err != nil {
		return parser.Option{}, err
	}
	return parser.Option{SkelIn: skelIn, SkelImports: imports}, nil
}

func normalizeGolangOption(option GolangOption) (golang.Option, error) {
	if strings.TrimSpace(option.Out) == "" {
		return golang.Option{}, optionValidationError("go.output", "required", "Go output is required")
	}
	if err := gomodule.ValidateVineVersion(option.VineVersion); err != nil {
		return golang.Option{}, err
	}
	modulePrefix := strings.TrimSpace(option.ModulePrefix)
	module := strings.TrimSpace(option.Module)
	pubOutValue := strings.TrimSpace(option.PubOut)
	pubModule := strings.TrimSpace(option.PubModule)
	if option.AsModule {
		if module == "" && modulePrefix == "" {
			return golang.Option{}, optionValidationError("go.module-identity", "required", "Go module or module prefix is required")
		}
	} else {
		invalidFields := []struct {
			field   string
			value   string
			message string
		}{
			{"go.public-output", pubOutValue, "Go public output requires module generation"},
			{"go.public-module", pubModule, "Go public module requires module generation"},
			{"go.module", module, "Go module requires module generation"},
			{"go.module-prefix", modulePrefix, "Go module prefix requires module generation"},
		}
		for _, field := range invalidFields {
			if field.value != "" {
				return golang.Option{}, optionValidationError(field.field, "requires-module", field.message)
			}
		}
	}
	if pubModule != "" && pubOutValue == "" {
		return golang.Option{}, optionValidationError("go.public-module", "requires-public-output", "Go public module requires public output")
	}

	moduleFields := []struct {
		value string
		name  string
		field string
	}{
		{modulePrefix, "Go module prefix", "go.module-prefix"},
		{module, "Go module", "go.module"},
		{pubModule, "Go public module", "go.public-module"},
	}
	for _, field := range moduleFields {
		if err := checkNoTrailingSlash(field.value, field.name, field.field); err != nil {
			return golang.Option{}, err
		}
	}
	out, err := absolutePath(option.Out)
	if err != nil {
		return golang.Option{}, err
	}
	if !option.AsModule {
		name := filepath.Base(out)
		if !nameutil.IsSnakeCase(name) || gotoken.Lookup(name).IsKeyword() {
			return golang.Option{}, fmt.Errorf("go output directory name %q is not a valid package name", name)
		}
	}
	pubOut, err := optionalAbsolutePath(pubOutValue)
	if err != nil {
		return golang.Option{}, err
	}
	imports, err := normalizeImportMap(option.Imports)
	if err != nil {
		return golang.Option{}, err
	}
	for _, path := range imports {
		if err := validateVersionedImport(path, "go"); err != nil {
			return golang.Option{}, err
		}
	}

	return golang.Option{
		CompilerVersion: strings.TrimSpace(option.CompilerVersion),
		AsModule:        option.AsModule,
		Out:             out,
		Module:          module,
		PubOut:          pubOut,
		PubModule:       pubModule,
		Imports:         imports,
		ModulePrefix:    modulePrefix,
		VineVersion:     strings.TrimSpace(option.VineVersion),
	}, nil
}

func validateGolangImports(domain *model.Domain, option golang.Option) error {
	for _, import_ := range domain.Imports() {
		if option.Imports[import_.Name] == "" && option.ModulePrefix == "" {
			return fmt.Errorf("missing Go import for domain %s; set Imports[%q] or ModulePrefix", import_.Name, import_.Name)
		}
	}
	return nil
}

func validateTypeScriptImports(domain *model.Domain, option typescript.Option) error {
	for _, import_ := range domain.Imports() {
		if option.Imports[import_.Name] == "" && option.ModuleScope == "" {
			return fmt.Errorf("missing TypeScript import for domain %s; set Imports[%q] or ModuleScope", import_.Name, import_.Name)
		}
	}
	return nil
}

func normalizeTypeScriptOption(option TypeScriptOption) (typescript.Option, error) {
	if strings.TrimSpace(option.Out) == "" {
		return typescript.Option{}, optionValidationError("typescript.output", "required", "TypeScript output is required")
	}
	moduleScope := strings.TrimRight(strings.TrimSpace(option.ModuleScope), "/")
	module := strings.TrimRight(strings.TrimSpace(option.Module), "/")
	if option.AsModule {
		if module == "" && moduleScope == "" {
			return typescript.Option{}, optionValidationError("typescript.module-identity", "required", "TypeScript module or module scope is required")
		}
	} else {
		if module != "" {
			return typescript.Option{}, optionValidationError("typescript.module", "requires-module", "TypeScript module requires module generation")
		}
		if moduleScope != "" {
			return typescript.Option{}, optionValidationError("typescript.module-scope", "requires-module", "TypeScript module scope requires module generation")
		}
	}
	out, err := absolutePath(option.Out)
	if err != nil {
		return typescript.Option{}, err
	}
	imports, err := normalizeTypeScriptImportMap(option.Imports)
	if err != nil {
		return typescript.Option{}, err
	}
	for _, path := range imports {
		if err := validateVersionedImport(path, "TypeScript"); err != nil {
			return typescript.Option{}, err
		}
	}

	return typescript.Option{
		PubOnly:     option.PubOnly,
		AsModule:    option.AsModule,
		Out:         out,
		Module:      module,
		Imports:     imports,
		ModuleScope: moduleScope,
	}, nil
}

func normalizeSkeletonOption(option SkeletonOption) (skeleton.Option, error) {
	if strings.TrimSpace(option.Out) == "" {
		return skeleton.Option{}, optionValidationError("skeleton.output", "required", "Skel output is required")
	}
	if !option.PubOnly {
		return skeleton.Option{}, optionValidationError("skeleton.public-only", "required", "Skel generation requires PubOnly")
	}
	out, err := absolutePath(option.Out)
	if err != nil {
		return skeleton.Option{}, err
	}
	return skeleton.Option{PubOnly: option.PubOnly, Out: out}, nil
}

func validateVersionedImport(path, kind string) error {
	index := strings.LastIndex(path, "@")
	if index < 0 {
		return nil
	}
	if index == 0 && kind == "go" {
		return fmt.Errorf("invalid %s import %q: missing module", kind, path)
	}
	if index == 0 {
		return nil
	}
	if index == len(path)-1 {
		return fmt.Errorf("invalid %s import %q: missing version", kind, path)
	}
	return nil
}

func checkNoTrailingSlash(value, label, field string) error {
	if strings.HasSuffix(value, "/") {
		return optionValidationError(field, "no-trailing-slash", label+" must not end with /")
	}
	return nil
}

func stageManagedOutputs(paths ...string) ([]*common.ManagedOutput, error) {
	outputs := make([]*common.ManagedOutput, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		output, err := common.NewManagedOutput(path)
		if err != nil {
			abortManagedOutputs(outputs)
			return nil, err
		}
		outputs = append(outputs, output)
	}
	return outputs, nil
}

func commitManagedOutputs(outputs []*common.ManagedOutput) error {
	for _, output := range outputs {
		if err := output.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func abortManagedOutputs(outputs []*common.ManagedOutput) {
	for _, output := range outputs {
		output.Abort()
	}
}

func absolutePath(path string) (string, error) {
	absPath, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", fmt.Errorf("resolve path %s: %w", path, err)
	}
	return absPath, nil
}

func optionalAbsolutePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	return absolutePath(path)
}

func normalizePathMap(values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := make(map[string]string, len(values))
	for _, key := range sortedMapKeys(values) {
		value := values[key]
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			return nil, fmt.Errorf("Skel import domain is required")
		}
		if strings.TrimSpace(value) == "" {
			return nil, fmt.Errorf("Skel import path for domain %s is required", normalizedKey)
		}
		path, err := absolutePath(value)
		if err != nil {
			return nil, err
		}
		if _, exists := normalized[normalizedKey]; exists {
			return nil, fmt.Errorf("duplicate Skel import domain %s", normalizedKey)
		}
		normalized[normalizedKey] = path
	}
	return normalized, nil
}

func normalizeImportMap(values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := make(map[string]string, len(values))
	for _, key := range sortedMapKeys(values) {
		value := values[key]
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			return nil, fmt.Errorf("Go import domain is required")
		}
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("Go import path for domain %s is required", normalizedKey)
		}
		if err := checkNoTrailingSlash(value, "Go import", "go.import"); err != nil {
			return nil, err
		}
		if _, exists := normalized[normalizedKey]; exists {
			return nil, fmt.Errorf("duplicate Go import domain %s", normalizedKey)
		}
		normalized[normalizedKey] = value
	}
	return normalized, nil
}

func normalizeTypeScriptImportMap(values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := make(map[string]string, len(values))
	for _, key := range sortedMapKeys(values) {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			return nil, fmt.Errorf("TypeScript import domain is required")
		}
		value := strings.TrimRight(strings.TrimSpace(values[key]), "/")
		if value == "" {
			return nil, fmt.Errorf("TypeScript import path for domain %s is required", normalizedKey)
		}
		if _, exists := normalized[normalizedKey]; exists {
			return nil, fmt.Errorf("duplicate TypeScript import domain %s", normalizedKey)
		}
		normalized[normalizedKey] = value
	}
	return normalized, nil
}

func sortedMapKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
