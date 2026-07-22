package parser

import (
	"path/filepath"
	"slices"

	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/model"
)

type CheckResult struct {
	Warnings    []string
	Diagnostics Diagnostics
}

// Check validates all discoverable source files and returns independent
// diagnostics. Unresolved imports remain allowed for compatibility with the
// check command, which does not accept import path mappings.
func Check(option Option) CheckResult {
	loadResult := loader.Load(option.SkelIn)
	expectedDomain := checkExpectedDomain(loadResult)
	sources := make([]Source, 0, len(loadResult.Files))
	for _, file := range loadResult.Files {
		sources = append(sources, Source{
			Path: file.FilePath, Domain: expectedDomain, ExpectedDomain: expectedDomain, Content: file.Content,
		})
	}
	structural := checkDirectoryStructure(loadResult)
	diagnostics := analyzeWorkspace(sources, true)
	filtered := make(Diagnostics, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if diagnostic.Code != DiagnosticCodeImport && (len(structural) == 0 || diagnostic.Code == DiagnosticCodeSyntax) {
			filtered = append(filtered, diagnostic)
		}
	}
	filtered = append(filtered, structural...)
	slices.SortFunc(filtered, compareDiagnostics)
	return CheckResult{Warnings: append([]string{}, loadResult.Warnings...), Diagnostics: filtered}
}

func checkDirectoryStructure(loadResult loader.Result) Diagnostics {
	if !loadResult.IsDir {
		return nil
	}
	diagnostics := Diagnostics{}
	for _, file := range loadResult.Files {
		content, err := ParseSource(file.FilePath, file.Content)
		if err != nil || content.Domain == nil {
			continue
		}
		position := model.Position{File: file.FilePath, Line: content.Pos.Line, Column: content.Pos.Column}
		if content.Domain.Name != nil {
			position = workspacePosition(content.Domain.Name.Pos)
		}
		if filepath.Base(file.FilePath) == loader.DomainFileName {
			if len(content.Entries) > 0 {
				diagnostics = append(diagnostics, Diagnostic{
					Code: DiagnosticCodeSemantic, Position: position,
					Message: file.FilePath + " can only contain domain declaration and @desc",
				})
			}
			continue
		}
		if len(content.Domain.Decorators) > 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Code: DiagnosticCodeSemantic, Position: position,
				Message: "domain decorator is only allowed in " + loader.DomainFileName + ": " + file.FilePath,
			})
		}
	}
	return diagnostics
}

func checkExpectedDomain(loadResult loader.Result) string {
	var domainFile *loader.SourceFile
	if loadResult.IsDir {
		for _, file := range loadResult.Files {
			if filepath.Base(file.FilePath) == loader.DomainFileName {
				domainFile = file
				break
			}
		}
	} else if len(loadResult.Files) > 0 {
		domainFile = loadResult.Files[0]
	}
	if domainFile == nil {
		return ""
	}
	content, err := ParseSource(domainFile.FilePath, domainFile.Content)
	if err != nil || content.Domain == nil || content.Domain.Name == nil {
		return ""
	}
	return content.Domain.Name.String()
}
