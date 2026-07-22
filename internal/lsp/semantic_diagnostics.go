package lsp

import (
	"path/filepath"

	"github.com/alecthomas/participle/v2/lexer"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/parser"
)

func semanticSources(documents map[uri.URI]*_Document) ([]parser.Source, map[string]uri.URI) {
	sources := make([]parser.Source, 0, len(documents))
	paths := make(map[string]uri.URI, len(documents))
	for documentURI, document := range documents {
		path := filepath.Clean(document.Path)
		sources = append(sources, parser.Source{
			Path: path, Domain: document.Domain, Content: []byte(document.Source),
		})
		paths[path] = documentURI
	}
	return sources, paths
}

func semanticDiagnostics(sources []parser.Source, paths map[string]uri.URI) map[uri.URI][]protocol.Diagnostic {
	result := map[uri.URI][]protocol.Diagnostic{}
	contents := make(map[string]string, len(sources))
	for _, source := range sources {
		contents[filepath.Clean(source.Path)] = string(source.Content)
	}
	for _, diagnostic := range parser.AnalyzeWorkspace(sources) {
		if diagnostic.Code == parser.DiagnosticCodeSyntax {
			continue
		}
		documentURI, ok := paths[filepath.Clean(diagnostic.Position.File)]
		if !ok {
			continue
		}
		source := contents[filepath.Clean(diagnostic.Position.File)]
		range_ := identifierRange(source, lexer.Position{
			Filename: diagnostic.Position.File, Line: diagnostic.Position.Line, Column: diagnostic.Position.Column,
		}, "")
		if range_.End == range_.Start {
			range_.End.Character++
		}
		result[documentURI] = append(result[documentURI], protocol.Diagnostic{
			Range: range_, Severity: protocol.DiagnosticSeverityError,
			Code: protocol.String(diagnostic.Code), Source: protocol.NewOptional("skelc"),
			Message: protocol.String(diagnostic.Message),
		})
	}
	return result
}
