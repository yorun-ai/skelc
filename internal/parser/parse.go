package parser

import (
	"path/filepath"
	"sort"

	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/hasher"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

type Option struct {
	SkelIn      string
	SkelImports map[string]string
}

type Result struct {
	Domain   *model.Domain
	Warnings []string
}

func Parse(option Option) Result {
	warnings := make([]string, 0)
	importedDomains := parseImportedDomains(option.SkelImports, &warnings)
	domain := parseSource(option.SkelIn, importedDomains, false, &warnings)
	warnings = append(warnings, domain.Warnings()...)
	sort.Strings(warnings)
	parsed := domain.Model()
	hasher.FillHashes(parsed)
	return Result{Domain: parsed, Warnings: warnings}
}

func ParseImport(skelIn string) Result {
	warnings := make([]string, 0)
	domain := parseSource(skelIn, nil, true, &warnings)
	warnings = append(warnings, domain.Warnings()...)
	sort.Strings(warnings)
	parsed := domain.Model()
	hasher.FillHashes(parsed)
	return Result{Domain: parsed, Warnings: warnings}
}

func parseImportedDomains(imports map[string]string, warnings *[]string) []*analyzer.Analysis {
	if len(imports) == 0 {
		return nil
	}
	domains := make([]*analyzer.Analysis, 0, len(imports))
	for expectedName, importPath := range imports {
		importedDomain := parseSource(importPath, nil, true, warnings)
		checkutil.Check(importedDomain.Model().Name() == expectedName,
			"skel import %s has domain %s", expectedName, importedDomain.Model().Name(),
		)
		domains = append(domains, importedDomain)
	}
	return domains
}

func parseSource(skelIn string, importedDomains []*analyzer.Analysis, importOnly bool, warnings *[]string) *analyzer.Analysis {
	loadResult := loader.Load(skelIn)
	for _, warning := range loadResult.Warnings {
		*warnings = append(*warnings, "[W] "+warning)
	}
	sourceParser := newParser()
	if !loadResult.IsDir {
		if importOnly {
			return sourceParser.parseImportFile(loadResult.Files[0])
		}
		return sourceParser.parseFileWithImports(loadResult.Files[0], importedDomains)
	}
	domainFile := findDomainFile(loadResult.Files)
	if importOnly {
		return sourceParser.parseImportFiles(domainFile, loadResult.Files)
	}
	return sourceParser.parseDomainFilesWithImports(domainFile, loadResult.Files, importedDomains)
}

func findDomainFile(sourceFiles []*loader.SourceFile) *loader.SourceFile {
	for _, sourceFile := range sourceFiles {
		if filepath.Base(sourceFile.FilePath) == loader.DomainFileName {
			return sourceFile
		}
	}
	checkutil.Failf("%s not found", loader.DomainFileName)
	return nil
}
