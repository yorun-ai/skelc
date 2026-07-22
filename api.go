// Package skelc provides the supported programmatic API for parsing Skel
// contracts and generating Go, TypeScript, and public Skel output.
//
// Use [Parse] when several generators or a custom generator need to share one
// validated [model.Domain]. The Compile functions combine parsing and one
// generation step for callers that only need a single target.
//
// Generation cleans its output directory by default. Set the target option's
// NoClean field only when the directory contains files that must be preserved.
package skelc

import (
	"go.yorun.ai/skelc/internal/codegen/golang"
	"go.yorun.ai/skelc/internal/codegen/skeleton"
	"go.yorun.ai/skelc/internal/codegen/typescript"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/util/checkutil"
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
	// Warnings contains non-fatal diagnostics produced while parsing.
	Warnings []string
}

// CompileResult contains non-fatal diagnostics produced while loading and parsing Skel sources.
type CompileResult struct {
	// Warnings contains non-fatal diagnostics produced while parsing.
	Warnings []string
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
	// NoClean preserves existing files in output directories before generation.
	NoClean bool
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
	// NoClean preserves existing files in the output directory before generation.
	NoClean bool
}

// SkeletonOption configures Skel source generation.
type SkeletonOption struct {
	// PubOnly limits output to declarations in the public contract.
	PubOnly bool
	// Out is the output directory for generated Skel files.
	Out string
	// NoClean preserves existing files in the output directory before generation.
	NoClean bool
}

// Parse loads and validates a Skel contract for use by custom generators and
// tools. Imported domains must be declared in Input.SkelImports.
func Parse(input Input) (result ParseResult, err error) {
	defer recoverAPIError(&err)

	parsed, parseErr := parser.Parse(normalizeInput(input))
	if parseErr != nil {
		return ParseResult{}, parseErr
	}
	return ParseResult{Domain: parsed.Domain, Warnings: parsed.Warnings}, nil
}

// GenerateGolang generates Go source or a standalone Go module from a parsed
// domain. Unless GolangOption.NoClean is set, it removes existing contents from
// each configured output directory before writing files.
func GenerateGolang(domain *model.Domain, option GolangOption) (err error) {
	defer recoverAPIError(&err)

	codegenOption := normalizeGolangOption(option)
	checkutil.CheckNotNil(domain, "parsed domain is required")
	prepareOutputDir(codegenOption.Out, option.NoClean)
	if codegenOption.PubOut != "" {
		prepareOutputDir(codegenOption.PubOut, option.NoClean)
	}
	golang.Generate(domain, codegenOption)
	return nil
}

// CompileGolang parses input and generates Go source or a standalone Go module.
// Parsing completes before any output directory is cleaned.
func CompileGolang(input Input, option GolangOption) (CompileResult, error) {
	parsed, err := Parse(input)
	if err != nil {
		return CompileResult{}, err
	}
	if err := GenerateGolang(parsed.Domain, option); err != nil {
		return CompileResult{}, err
	}
	return CompileResult{Warnings: parsed.Warnings}, nil
}

// GenerateTypeScript generates TypeScript source from a parsed domain. Unless
// TypeScriptOption.NoClean is set, it removes existing contents from the output
// directory before writing files.
func GenerateTypeScript(domain *model.Domain, option TypeScriptOption) (err error) {
	defer recoverAPIError(&err)

	codegenOption := normalizeTypeScriptOption(option)
	checkutil.CheckNotNil(domain, "parsed domain is required")
	prepareOutputDir(codegenOption.Out, option.NoClean)
	typescript.Generate(domain, codegenOption)
	return nil
}

// CompileTypeScript parses input and generates TypeScript source. Parsing
// completes before the output directory is cleaned.
func CompileTypeScript(input Input, option TypeScriptOption) (CompileResult, error) {
	parsed, err := Parse(input)
	if err != nil {
		return CompileResult{}, err
	}
	if err := GenerateTypeScript(parsed.Domain, option); err != nil {
		return CompileResult{}, err
	}
	return CompileResult{Warnings: parsed.Warnings}, nil
}

// GenerateSkeleton generates a Skel contract from a parsed domain. Unless
// SkeletonOption.NoClean is set, it removes existing contents from the output
// directory before writing files.
func GenerateSkeleton(domain *model.Domain, option SkeletonOption) (err error) {
	defer recoverAPIError(&err)

	codegenOption := normalizeSkeletonOption(option)
	checkutil.CheckNotNil(domain, "parsed domain is required")
	prepareOutputDir(codegenOption.Out, option.NoClean)
	skeleton.Generate(domain, codegenOption)
	return nil
}

// CompileSkeleton parses input and generates a Skel contract. Parsing completes
// before the output directory is cleaned.
func CompileSkeleton(input Input, option SkeletonOption) (CompileResult, error) {
	parsed, err := Parse(input)
	if err != nil {
		return CompileResult{}, err
	}
	if err := GenerateSkeleton(parsed.Domain, option); err != nil {
		return CompileResult{}, err
	}
	return CompileResult{Warnings: parsed.Warnings}, nil
}
