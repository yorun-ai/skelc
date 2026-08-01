package compiler

import (
	"errors"
	"fmt"
	"path/filepath"

	"go.yorun.ai/skelc/internal/analyzer"
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

func parseImportFile(sourceFile *loader.SourceFile) (*analyzer.Analysis, error) {
	content, err := parseContent(sourceFile, true)
	if err != nil {
		return nil, err
	}
	analysis, diagnostics := analyzer.AnalyzeImport(content)
	return analysis, errors.Join(diagnostics...)
}

func parseFileWithImports(sourceFile *loader.SourceFile, importedDomains []*analyzer.Analysis) (*analyzer.Analysis, error) {
	content, err := parseContent(sourceFile, true)
	if err != nil {
		return nil, err
	}
	analysis, diagnostics := analyzer.Analyze(content, importedDomains)
	return analysis, errors.Join(diagnostics...)
}

func parseImportFiles(domainFile *loader.SourceFile, inputFiles []*loader.SourceFile) (*analyzer.Analysis, error) {
	domainFileContent, err := parseDomainFile(domainFile)
	if err != nil {
		return nil, err
	}
	parsedContents, err := parseContentsExcept(inputFiles, domainFile.FilePath)
	if err != nil {
		return nil, err
	}
	domainName := domainFileContent.Domain.Name.String()
	if err := validateDirectoryDomains(domainName, parsedContents); err != nil {
		return nil, err
	}
	analysis, diagnostics := analyzer.AnalyzeImport(mergeDomainContents(append([]*grammar.SkelContent{domainFileContent}, parsedContents...)))
	return analysis, errors.Join(diagnostics...)
}

func parseDomainFilesWithImports(domainFile *loader.SourceFile, inputFiles []*loader.SourceFile, importedDomains []*analyzer.Analysis) (*analyzer.Analysis, error) {
	domainFileContent, err := parseDomainFile(domainFile)
	if err != nil {
		return nil, err
	}
	parsedContents, err := parseContentsExcept(inputFiles, domainFile.FilePath)
	if err != nil {
		return nil, err
	}
	domainName := domainFileContent.Domain.Name.String()
	if err := validateDirectoryDomains(domainName, parsedContents); err != nil {
		return nil, err
	}
	analysis, diagnostics := analyzer.Analyze(mergeDomainContents(append([]*grammar.SkelContent{domainFileContent}, parsedContents...)), importedDomains)
	return analysis, errors.Join(diagnostics...)
}

func parseDomainFile(domainFile *loader.SourceFile) (*grammar.SkelContent, error) {
	content, err := parseContent(domainFile, true)
	if err != nil {
		return nil, err
	}
	if issue := inspectDirectorySource(domainFile.FilePath, content.Domain.Name.String(), content); issue != nil {
		return nil, errors.New(issue.strictMessage)
	}
	return content, nil
}

func parseContentsExcept(inputFiles []*loader.SourceFile, excludedPath string) ([]*grammar.SkelContent, error) {
	contents := make([]*grammar.SkelContent, 0, len(inputFiles))
	for _, inputFile := range inputFiles {
		if excludedPath != "" && filepath.Clean(inputFile.FilePath) == filepath.Clean(excludedPath) {
			continue
		}
		content, err := parseContent(inputFile, true)
		if err != nil {
			return nil, err
		}
		contents = append(contents, content)
	}
	return contents, nil
}

func validateDirectoryDomains(domainName string, contents []*grammar.SkelContent) error {
	for _, content := range contents {
		if issue := inspectDirectorySource(content.Pos.Filename, domainName, content); issue != nil {
			return errors.New(issue.strictMessage)
		}
	}
	return nil
}

func parseContent(sourceFile *loader.SourceFile, requireDomain bool) (*grammar.SkelContent, error) {
	content, err := parser.ParseSource(sourceFile.FilePath, sourceFile.Content)
	if err != nil {
		return nil, fmt.Errorf("parse %s failed: %w", sourceFile.FilePath, err)
	}

	if content.Domain == nil || content.Domain.Name == nil {
		if requireDomain {
			return nil, fmt.Errorf("missing domain declaration in %s", sourceFile.FilePath)
		}
		return content, nil
	}
	if content.Domain.Name.String() == "" {
		return nil, fmt.Errorf("missing domain name in %s", sourceFile.FilePath)
	}
	return content, nil
}
