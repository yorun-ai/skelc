package parser

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/model"
)

type CheckResult struct {
	Diagnostics Diagnostics
}

// Check validates all discoverable source files and returns independent
// diagnostics. Unresolved imports remain allowed for compatibility with the
// check command, which does not accept import path mappings.
func Check(option Option) (CheckResult, error) {
	return checkWithAnalyzer(option, NewWorkspaceAnalyzer())
}

func checkWithAnalyzer(option Option, workspaceAnalyzer *WorkspaceAnalyzer) (CheckResult, error) {
	loadResult, err := loader.Load(option.SkelIn)
	if err != nil {
		return CheckResult{}, err
	}
	sources := parseCheckSources(loadResult.Files)
	expectedDomain := checkExpectedDomain(loadResult, sources)
	for index := range sources {
		sources[index].Domain = expectedDomain
		sources[index].ExpectedDomain = expectedDomain
	}
	structural := checkDirectoryStructure(loadResult, sources)
	diagnostics, err := workspaceAnalyzer.analyze(context.Background(), sources, true)
	if err != nil {
		return CheckResult{}, err
	}
	filtered := make(Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != DiagnosticCodeImportMissing && (len(structural) == 0 || strings.HasPrefix(diagnostic.Code, "syntax.")) {
			filtered = append(filtered, diagnostic)
		}
	}
	filtered = append(filtered, structural...)
	filtered = append(filtered, loaderWarningDiagnostics(loadResult.Warnings)...)
	slices.SortFunc(filtered, compareDiagnostics)
	return CheckResult{Diagnostics: filtered}, nil
}

func parseCheckSources(files []*loader.SourceFile) []Source {
	sources := make([]Source, 0, len(files))
	for _, file := range files {
		content, diagnostics := ParseSourceRecovering(file.FilePath, file.Content)
		sources = append(sources, Source{
			Path: file.FilePath, Content: file.Content, Parsed: content, ParseDiagnostics: diagnostics,
		})
	}
	return sources
}

func checkDirectoryStructure(loadResult loader.Result, sources []Source) Diagnostics {
	if !loadResult.IsDir {
		return nil
	}
	diagnostics := Diagnostics{}
	for _, source := range sources {
		content := source.Parsed
		if len(source.ParseDiagnostics) > 0 || content == nil || content.Domain == nil {
			continue
		}
		position := model.Position{File: source.Path, Line: content.Pos.Line, Column: content.Pos.Column}
		if content.Domain.Name != nil {
			position = workspacePosition(content.Domain.Name.Pos)
		}
		if filepath.Base(source.Path) == loader.DomainFileName {
			if len(content.Entries) > 0 {
				diagnostics = append(diagnostics, Diagnostic{
					Code: DiagnosticCodeDomainFileContent, Severity: DiagnosticSeverityError, Position: position,
					Range:   sourceRangeAt(position, source.Content),
					Message: source.Path + " can only contain domain declaration and @desc",
				})
			}
			continue
		}
		if len(content.Domain.Decorators) > 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Code: DiagnosticCodeDomainDecorator, Severity: DiagnosticSeverityError, Position: position,
				Range:   sourceRangeAt(position, source.Content),
				Message: "domain decorator is only allowed in " + loader.DomainFileName + ": " + source.Path,
			})
		}
	}
	return diagnostics
}

func checkExpectedDomain(loadResult loader.Result, sources []Source) string {
	var domainSource *Source
	if loadResult.IsDir {
		for index := range sources {
			if filepath.Base(sources[index].Path) == loader.DomainFileName {
				domainSource = &sources[index]
				break
			}
		}
	} else if len(sources) > 0 {
		domainSource = &sources[0]
	}
	if domainSource == nil {
		return ""
	}
	content := domainSource.Parsed
	if len(domainSource.ParseDiagnostics) > 0 || content == nil || content.Domain == nil || content.Domain.Name == nil {
		return ""
	}
	return content.Domain.Name.String()
}
