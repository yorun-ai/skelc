package parser

import (
	"bytes"
	"path/filepath"

	"github.com/alecthomas/participle/v2"
	"go.yorun.ai/skelc/internal/loader"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

const domainFileName = "domain.skel"

type _Parser struct {
	skelContentParser *participle.Parser[grammar.SkelContent]
}

// ParseSource parses and finalizes one Skel source file without resolving its
// imports or performing domain-level semantic analysis.
func ParseSource(path string, source []byte) (*grammar.SkelContent, error) {
	parser := newParser()
	content, err := parser.skelContentParser.Parse(path, bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	if err := content.Finalize(); err != nil {
		return nil, err
	}
	if content.Domain != nil {
		if err := content.Domain.Finalize(); err != nil {
			return nil, err
		}
	}
	return content, nil
}

// ValidateSource validates the grammar and finalized syntax state of one Skel source file.
func ValidateSource(path string, source []byte) {
	_, err := ParseSource(path, source)
	checkutil.CheckNilError(err, "parse %s failed", path)
}

func newParser() *_Parser {
	return &_Parser{
		skelContentParser: participle.MustBuild[grammar.SkelContent](grammar.Options...),
	}
}

func (p *_Parser) parseFile(sourceFile *loader.SourceFile) *analyzer.Analysis {
	return p.parseFileWithImports(sourceFile, nil)
}

func (p *_Parser) parseImportFile(sourceFile *loader.SourceFile) *analyzer.Analysis {
	content := p.parseContent(sourceFile, true)
	return analyzer.AnalyzeImport(content)
}

func (p *_Parser) parseFileWithImports(sourceFile *loader.SourceFile, importedDomains []*analyzer.Analysis) *analyzer.Analysis {
	content := p.parseContent(sourceFile, true)
	return analyzer.AnalyzeWithImports(content, importedDomains)
}

func (p *_Parser) parseDomainFiles(domainFile *loader.SourceFile, inputFiles []*loader.SourceFile) *analyzer.Analysis {
	return p.parseDomainFilesWithImports(domainFile, inputFiles, nil)
}

func (p *_Parser) parseImportFiles(domainFile *loader.SourceFile, inputFiles []*loader.SourceFile) *analyzer.Analysis {
	domainFileContent := p.parseDomainFile(domainFile)
	parsedContents := p.parseContents(inputFiles)
	domainName := domainFileContent.Domain.Name.String()
	p.validateDirectoryDomains(domainName, parsedContents)
	return analyzer.AnalyzeImport(buildMergedContent(domainFileContent, parsedContents))
}

func (p *_Parser) parseDomainFilesWithImports(domainFile *loader.SourceFile, inputFiles []*loader.SourceFile, importedDomains []*analyzer.Analysis) *analyzer.Analysis {
	domainFileContent := p.parseDomainFile(domainFile)
	parsedContents := p.parseContents(inputFiles)
	domainName := domainFileContent.Domain.Name.String()
	p.validateDirectoryDomains(domainName, parsedContents)
	return analyzer.AnalyzeWithImports(buildMergedContent(domainFileContent, parsedContents), importedDomains)
}

func (p *_Parser) parseDomainFile(domainFile *loader.SourceFile) *grammar.SkelContent {
	return p.parseContent(domainFile, true)
}

func (p *_Parser) parseContents(inputFiles []*loader.SourceFile) []*grammar.SkelContent {
	contents := make([]*grammar.SkelContent, 0, len(inputFiles))
	for _, inputFile := range inputFiles {
		content := p.parseContent(inputFile, true)
		contents = append(contents, content)
	}
	return contents
}

func (p *_Parser) validateDirectoryDomains(domainName string, contents []*grammar.SkelContent) {
	for _, content := range contents {
		if filepath.Base(content.Pos.Filename) == domainFileName {
			checkutil.Check(content.Domain != nil && content.Domain.Name != nil,
				"missing domain declaration in %s", content.Pos.Filename)
			checkutil.Check(content.Domain.Name.String() == domainName,
				"domain mismatch in %s: found=%s, expected=%s", content.Pos.Filename, content.Domain.Name.String(), domainName)
			checkutil.Check(len(content.Entries) == 0, "%s can only contain domain declaration and @desc", content.Pos.Filename)
			continue
		}
		checkutil.Check(content.Domain != nil && content.Domain.Name != nil,
			"missing domain declaration in %s", content.Pos.Filename)
		checkutil.Check(content.Domain.Name.String() == domainName,
			"domain mismatch in %s: found=%s, expected=%s", content.Pos.Filename, content.Domain.Name.String(), domainName)
		checkutil.Check(len(content.Domain.Decorators) == 0,
			"domain decorator is only allowed in %s: %s", domainFileName, content.Pos.Filename)
	}
}

func buildMergedContent(domainFileContent *grammar.SkelContent, contents []*grammar.SkelContent) *grammar.SkelContent {
	merged := &grammar.SkelContent{
		Domain:  domainFileContent.Domain,
		Imports: make([]*grammar.ImportDecl, 0),
		Entries: make([]*grammar.SkelEntry, 0),
	}
	for _, content := range contents {
		merged.Imports = append(merged.Imports, content.Imports...)
		merged.Entries = append(merged.Entries, content.Entries...)
	}
	return merged
}

func (p *_Parser) parseContent(sourceFile *loader.SourceFile, requireDomain bool) *grammar.SkelContent {
	content, err := ParseSource(sourceFile.FilePath, sourceFile.Content)
	checkutil.CheckNilError(err, "parse %s failed", sourceFile.FilePath)

	if content.Domain == nil || content.Domain.Name == nil {
		checkutil.CheckNot(requireDomain, "missing domain declaration in %s", sourceFile.FilePath)
		return content
	}
	checkutil.Check(content.Domain.Name.String() != "", "missing domain name in %s", sourceFile.FilePath)
	return content
}
