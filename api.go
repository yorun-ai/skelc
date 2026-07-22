// Package skelc provides the supported programmatic API for parsing Skel
// contracts and generating Go, TypeScript, and public Skel output.
//
// Use [Parse] when several generators or a custom generator need to share one
// validated [model.Domain]. The Compile functions combine parsing and one
// generation step for callers that only need a single target.
//
// Generation tracks its own files in an output manifest. Existing files that
// are not owned by skelc are preserved.
package skelc

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/codegen/skeleton"
	"go.yorun.ai/skelc/internal/codegen/typescript"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/model"
)

// DefaultGolangVineVersion is the Vine module version used for generated Go
// modules when GolangOption.VineVersion is empty.
const DefaultGolangVineVersion = golang.DefaultVineVersion

// Input identifies the primary Skel source and any imported domains.
type Input struct {
	// SkelIn is the path to a Skel source file or domain directory.
	SkelIn string
	// SkelImports maps imported domain names to Skel source files or directories.
	SkelImports map[string]string
}

// ParseResult contains a validated semantic model and non-fatal diagnostics.
type ParseResult struct {
	// Domain is the validated semantic model, including compatibility hashes.
	Domain *model.Domain
	// Diagnostics contains structured non-fatal diagnostics produced while parsing.
	Diagnostics []Diagnostic
}

type Diagnostic = parser.Diagnostic
type DiagnosticSeverity = parser.DiagnosticSeverity
type SourceRange = parser.SourceRange
type DiagnosticRelatedInformation = parser.DiagnosticRelatedInformation
type DiagnosticSuggestion = parser.DiagnosticSuggestion

const (
	DiagnosticSeverityError   = parser.DiagnosticSeverityError
	DiagnosticSeverityWarning = parser.DiagnosticSeverityWarning
)

// CompileResult contains structured non-fatal diagnostics produced while loading and parsing Skel sources.
type CompileResult struct {
	// Diagnostics contains non-fatal diagnostics produced while parsing.
	Diagnostics []Diagnostic
}

// GolangOption configures Go generation.
type GolangOption struct {
	// CompilerVersion identifies the skelc version embedded in generated metadata.
	CompilerVersion string
	// AsModule generates a standalone Go module instead of package source for an
	// existing module.
	AsModule bool
	// Out is the output directory for generated Go files.
	Out string
	// Module is the module path used when AsModule is true.
	Module string
	// PubOut is the optional output directory for a separate public Go module.
	PubOut string
	// PubModule is the module path for PubOut.
	PubModule string
	// Imports maps imported Skel domain names to Go import paths.
	Imports map[string]string
	// ModulePrefix derives module paths from domain names when Module or an
	// imported-domain mapping is omitted.
	ModulePrefix string
	// VineVersion selects the go.yorun.ai/vine version written to generated module
	// metadata. An empty value uses [DefaultGolangVineVersion].
	VineVersion string
}

// TypeScriptOption configures TypeScript generation.
type TypeScriptOption struct {
	// PubOnly limits output to the domain's public contract.
	PubOnly bool
	// AsModule emits package metadata for a standalone npm package.
	AsModule bool
	// Out is the output directory for generated TypeScript files.
	Out string
	// Module is the npm package name used when AsModule is true.
	Module string
	// Imports maps imported Skel domain names to npm package specifiers.
	Imports map[string]string
	// ModuleScope derives npm package names for the current and imported domains.
	ModuleScope string
}

// SkeletonOption configures Skel source generation.
type SkeletonOption struct {
	// PubOnly limits output to declarations in the public contract.
	PubOnly bool
	// Out is the output directory for generated Skel files.
	Out string
}

// Parse loads and validates a Skel contract for use by custom generators and
// tools. Imported domains must be declared in Input.SkelImports.
func Parse(input Input) (ParseResult, error) {
	option, err := normalizeInput(input)
	if err != nil {
		return ParseResult{}, err
	}
	parsed, parseErr := parser.Parse(option)
	if parseErr != nil {
		return ParseResult{}, parseErr
	}
	return ParseResult{Domain: parsed.Domain, Diagnostics: parsed.Diagnostics}, nil
}

// GenerateGolang generates Go source or a standalone Go module from a parsed
// domain. Only files owned by the previous skelc manifest may be removed.
func GenerateGolang(domain *model.Domain, option GolangOption) error {
	if domain == nil {
		return fmt.Errorf("parsed domain is required")
	}
	codegenOption, err := normalizeGolangOption(option)
	if err != nil {
		return err
	}
	return generateGolang(domain, codegenOption)
}

func generateGolang(domain *model.Domain, option golang.Option) error {
	if err := validateGolangImports(domain, option); err != nil {
		return err
	}
	outputs, outputErr := stageManagedOutputs(option.Out, option.PubOut)
	if outputErr != nil {
		return outputErr
	}
	defer abortManagedOutputs(outputs)
	option.Out = outputs[0].StageDir()
	if option.PubOut != "" {
		option.PubOut = outputs[1].StageDir()
	}
	if err := golang.Generate(domain, option); err != nil {
		return err
	}
	return commitManagedOutputs(outputs)
}

// CompileGolang parses input and generates Go source or a standalone Go module.
// Parsing completes before any generated output is committed.
func CompileGolang(input Input, option GolangOption) (CompileResult, error) {
	parserOption, err := normalizeInput(input)
	if err != nil {
		return CompileResult{}, err
	}
	codegenOption, err := normalizeGolangOption(option)
	if err != nil {
		return CompileResult{}, err
	}
	parsed, err := parser.Parse(parserOption)
	if err != nil {
		return CompileResult{}, err
	}
	if err := generateGolang(parsed.Domain, codegenOption); err != nil {
		return CompileResult{}, err
	}
	return CompileResult{Diagnostics: parsed.Diagnostics}, nil
}

// GenerateTypeScript generates TypeScript source from a parsed domain. Only
// files owned by the previous skelc manifest may be removed.
func GenerateTypeScript(domain *model.Domain, option TypeScriptOption) error {
	if domain == nil {
		return fmt.Errorf("parsed domain is required")
	}
	codegenOption, err := normalizeTypeScriptOption(option)
	if err != nil {
		return err
	}
	return generateTypeScript(domain, codegenOption)
}

func generateTypeScript(domain *model.Domain, option typescript.Option) error {
	if err := validateTypeScriptImports(domain, option); err != nil {
		return err
	}
	outputs, outputErr := stageManagedOutputs(option.Out)
	if outputErr != nil {
		return outputErr
	}
	defer abortManagedOutputs(outputs)
	option.Out = outputs[0].StageDir()
	if err := typescript.Generate(domain, option); err != nil {
		return err
	}
	return commitManagedOutputs(outputs)
}

// CompileTypeScript parses input and generates TypeScript source. Parsing
// completes before generated output is committed.
func CompileTypeScript(input Input, option TypeScriptOption) (CompileResult, error) {
	parserOption, err := normalizeInput(input)
	if err != nil {
		return CompileResult{}, err
	}
	codegenOption, err := normalizeTypeScriptOption(option)
	if err != nil {
		return CompileResult{}, err
	}
	parsed, err := parser.Parse(parserOption)
	if err != nil {
		return CompileResult{}, err
	}
	if err := generateTypeScript(parsed.Domain, codegenOption); err != nil {
		return CompileResult{}, err
	}
	return CompileResult{Diagnostics: parsed.Diagnostics}, nil
}

// GenerateSkeleton generates a Skel contract from a parsed domain. Only files
// owned by the previous skelc manifest may be removed.
func GenerateSkeleton(domain *model.Domain, option SkeletonOption) error {
	if domain == nil {
		return fmt.Errorf("parsed domain is required")
	}
	codegenOption, err := normalizeSkeletonOption(option)
	if err != nil {
		return err
	}
	return generateSkeleton(domain, codegenOption)
}

func generateSkeleton(domain *model.Domain, option skeleton.Option) error {
	outputs, outputErr := stageManagedOutputs(option.Out)
	if outputErr != nil {
		return outputErr
	}
	defer abortManagedOutputs(outputs)
	option.Out = outputs[0].StageDir()
	if err := skeleton.Generate(domain, option); err != nil {
		return err
	}
	return commitManagedOutputs(outputs)
}

// CompileSkeleton parses input and generates a Skel contract. Parsing completes
// before generated output is committed.
func CompileSkeleton(input Input, option SkeletonOption) (CompileResult, error) {
	parserOption, err := normalizeInput(input)
	if err != nil {
		return CompileResult{}, err
	}
	codegenOption, err := normalizeSkeletonOption(option)
	if err != nil {
		return CompileResult{}, err
	}
	parsed, err := parser.Parse(parserOption)
	if err != nil {
		return CompileResult{}, err
	}
	if err := generateSkeleton(parsed.Domain, codegenOption); err != nil {
		return CompileResult{}, err
	}
	return CompileResult{Diagnostics: parsed.Diagnostics}, nil
}
