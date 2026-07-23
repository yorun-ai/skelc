package parser

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/hasher"
	"go.yorun.ai/skelc/model"
)

type Option struct {
	SkelIn      string
	SkelImports map[string]string
}

type Result struct {
	Domain      *model.Domain
	Diagnostics Diagnostics
}

func Parse(option Option) (Result, error) {
	diagnostics := Diagnostics{}
	importedDomains, err := parseImportedDomains(option.SkelImports, &diagnostics)
	if err != nil {
		return Result{}, err
	}
	domain, err := parseSource(option.SkelIn, importedDomains, false, &diagnostics)
	if err != nil {
		return Result{}, err
	}
	diagnostics = appendAnalysisWarnings(diagnostics, domain.Warnings())
	slices.SortFunc(diagnostics, compareDiagnostics)
	parsed := domain.Model()
	hasher.FillHashes(parsed)
	return Result{Domain: parsed, Diagnostics: diagnostics}, nil
}

// ParseImport loads one domain without requiring its dependencies. It supports
// symbol tooling; Parse and Compile APIs perform complete graph analysis.
func ParseImport(skelIn string) (Result, error) {
	diagnostics := Diagnostics{}
	domain, err := parseSource(skelIn, nil, true, &diagnostics)
	if err != nil {
		return Result{}, err
	}
	diagnostics = appendAnalysisWarnings(diagnostics, domain.Warnings())
	slices.SortFunc(diagnostics, compareDiagnostics)
	parsed := domain.Model()
	hasher.FillHashes(parsed)
	return Result{Domain: parsed, Diagnostics: diagnostics}, nil
}

func parseImportedDomains(imports map[string]string, diagnostics *Diagnostics) ([]*analyzer.Analysis, error) {
	if len(imports) == 0 {
		return nil, nil
	}
	loaded := make(map[string]*analyzer.Analysis, len(imports))
	for expectedName, importPath := range imports {
		importedDomain, err := parseSource(importPath, nil, true, diagnostics)
		if err != nil {
			return nil, err
		}
		if importedDomain.Model().Name() != expectedName {
			return nil, fmt.Errorf("skel import %s has domain %s", expectedName, importedDomain.Model().Name())
		}
		loaded[expectedName] = importedDomain
	}

	const (
		importVisiting = iota + 1
		importComplete
	)
	states := make(map[string]int, len(loaded))
	resolved := make(map[string]*analyzer.Analysis, len(loaded))
	var resolve func(string) (*analyzer.Analysis, error)
	resolve = func(name string) (*analyzer.Analysis, error) {
		switch states[name] {
		case importVisiting:
			return nil, fmt.Errorf("cyclic skel import involving %s", name)
		case importComplete:
			return resolved[name], nil
		}

		states[name] = importVisiting
		domain := loaded[name]
		dependencies := make([]*analyzer.Analysis, 0, len(domain.ImportNames()))
		for _, dependencyName := range domain.ImportNames() {
			if loaded[dependencyName] == nil {
				continue
			}
			dependency, err := resolve(dependencyName)
			if err != nil {
				return nil, err
			}
			dependencies = append(dependencies, dependency)
		}
		domain, analysisErrors := domain.ResolveImports(dependencies)
		if len(analysisErrors) > 0 {
			return nil, errors.Join(analysisErrors...)
		}
		resolved[name] = domain
		states[name] = importComplete
		return domain, nil
	}

	names := make([]string, 0, len(loaded))
	for name := range loaded {
		names = append(names, name)
	}
	slices.Sort(names)
	domains := make([]*analyzer.Analysis, 0, len(names))
	for _, name := range names {
		domain, err := resolve(name)
		if err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}
	return domains, nil
}

func parseSource(skelIn string, importedDomains []*analyzer.Analysis, importOnly bool, diagnostics *Diagnostics) (*analyzer.Analysis, error) {
	loadResult, err := loader.Load(skelIn)
	if err != nil {
		return nil, err
	}
	*diagnostics = append(*diagnostics, loaderWarningDiagnostics(loadResult.Warnings)...)
	if !loadResult.IsDir {
		if importOnly {
			return parseImportFile(loadResult.Files[0])
		}
		return parseFileWithImports(loadResult.Files[0], importedDomains)
	}
	domainFile, err := findDomainFile(loadResult.Files)
	if err != nil {
		return nil, err
	}
	if importOnly {
		return parseImportFiles(domainFile, loadResult.Files)
	}
	return parseDomainFilesWithImports(domainFile, loadResult.Files, importedDomains)
}

func loaderWarningDiagnostics(warnings []loader.Warning) Diagnostics {
	diagnostics := make(Diagnostics, 0, len(warnings))
	for _, warning := range warnings {
		position := model.Position{File: warning.Path, Line: 1, Column: 1}
		diagnostics = append(diagnostics, Diagnostic{
			Code: warning.Code, Severity: DiagnosticSeverityWarning, Position: position,
			Range: SourceRange{Start: position, End: position}, Message: warning.Message,
		})
	}
	return diagnostics
}

func appendAnalysisWarnings(diagnostics Diagnostics, warnings []string) Diagnostics {
	for _, warning := range warnings {
		diagnostics = append(diagnostics, Diagnostic{
			Code: DiagnosticCodeSemanticWarning, Severity: DiagnosticSeverityWarning, Message: warning,
		})
	}
	return diagnostics
}

func findDomainFile(sourceFiles []*loader.SourceFile) (*loader.SourceFile, error) {
	for _, sourceFile := range sourceFiles {
		if filepath.Base(sourceFile.FilePath) == loader.DomainFileName {
			return sourceFile, nil
		}
	}
	return nil, fmt.Errorf("%s not found", loader.DomainFileName)
}
