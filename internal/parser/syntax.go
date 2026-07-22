package parser

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/alecthomas/participle/v2"
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

const domainFileName = "domain.skel"

var sourceParser = participle.MustBuild[grammar.SkelContent](grammar.Options...)

// ParseSource parses and finalizes one Skel source file without resolving its
// imports or performing domain-level semantic analysis.
func ParseSource(path string, source []byte) (*grammar.SkelContent, error) {
	content, err, _ := parseSourceOnce(path, source)
	return content, err
}

// ValidateSource validates the grammar and finalized syntax state of one Skel source file.
func ValidateSource(path string, source []byte) error {
	_, err := ParseSource(path, source)
	if err != nil {
		return fmt.Errorf("parse %s failed: %w", path, err)
	}
	return nil
}

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
	analysis, diagnostics := analyzer.AnalyzeImport(buildMergedContent(domainFileContent, parsedContents))
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
	analysis, diagnostics := analyzer.Analyze(buildMergedContent(domainFileContent, parsedContents), importedDomains)
	return analysis, errors.Join(diagnostics...)
}

func parseDomainFile(domainFile *loader.SourceFile) (*grammar.SkelContent, error) {
	content, err := parseContent(domainFile, true)
	if err != nil {
		return nil, err
	}
	if len(content.Entries) != 0 {
		return nil, fmt.Errorf("%s can only contain domain declaration and @desc", content.Pos.Filename)
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
		if filepath.Base(content.Pos.Filename) == domainFileName {
			if content.Domain == nil || content.Domain.Name == nil {
				return fmt.Errorf("missing domain declaration in %s", content.Pos.Filename)
			}
			if content.Domain.Name.String() != domainName {
				return fmt.Errorf("domain mismatch in %s: found=%s, expected=%s", content.Pos.Filename, content.Domain.Name.String(), domainName)
			}
			if len(content.Entries) != 0 {
				return fmt.Errorf("%s can only contain domain declaration and @desc", content.Pos.Filename)
			}
			continue
		}
		if content.Domain == nil || content.Domain.Name == nil {
			return fmt.Errorf("missing domain declaration in %s", content.Pos.Filename)
		}
		if content.Domain.Name.String() != domainName {
			return fmt.Errorf("domain mismatch in %s: found=%s, expected=%s", content.Pos.Filename, content.Domain.Name.String(), domainName)
		}
		if len(content.Domain.Decorators) != 0 {
			return fmt.Errorf("domain decorator is only allowed in %s: %s", domainFileName, content.Pos.Filename)
		}
	}
	return nil
}

func buildMergedContent(domainFileContent *grammar.SkelContent, contents []*grammar.SkelContent) *grammar.SkelContent {
	merged := &grammar.SkelContent{
		Domain:  domainFileContent.Domain,
		Imports: append([]*grammar.ImportDecl{}, domainFileContent.Imports...),
		Entries: make([]*grammar.SkelEntry, 0),
	}
	for _, content := range contents {
		if content == domainFileContent {
			continue
		}
		merged.Imports = append(merged.Imports, content.Imports...)
		merged.Entries = append(merged.Entries, content.Entries...)
	}
	return merged
}

func parseContent(sourceFile *loader.SourceFile, requireDomain bool) (*grammar.SkelContent, error) {
	content, err := ParseSource(sourceFile.FilePath, sourceFile.Content)
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
