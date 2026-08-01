package compiler

import (
	"context"
	"path/filepath"
	"slices"
	"strings"

	"go.yorun.ai/skelc/internal/loader"
)

// CheckResult contains the diagnostics produced by a check operation.
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
	structural := checkDirectoryStructure(loadResult, sources, expectedDomain)
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

func checkDirectoryStructure(loadResult loader.Result, sources []Source, expectedDomain string) Diagnostics {
	if !loadResult.IsDir {
		return nil
	}
	diagnostics := Diagnostics{}
	for _, source := range sources {
		content := source.Parsed
		if len(source.ParseDiagnostics) > 0 || content == nil {
			continue
		}
		if issue := inspectDirectorySource(source.Path, expectedDomain, content); issue != nil {
			diagnostics = append(diagnostics, Diagnostic{
				Code: issue.code, Severity: DiagnosticSeverityError, Position: issue.position,
				Range: sourceRangeAt(issue.position, source.Content), Message: issue.message,
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
