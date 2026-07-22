package lsp

import (
	"context"
	"path/filepath"
	"strings"

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
			Path: path, Domain: document.Domain, Content: []byte(document.Source), Parsed: document.Parsed,
			ParseDiagnostics: document.ParseDiagnostics,
		})
		paths[path] = documentURI
	}
	return sources, paths
}

func semanticDiagnostics(ctx context.Context, analyzer *parser.WorkspaceAnalyzer, sources []parser.Source, paths map[string]uri.URI) (map[uri.URI][]protocol.Diagnostic, error) {
	result := map[uri.URI][]protocol.Diagnostic{}
	contents := make(map[string]string, len(sources))
	for _, source := range sources {
		contents[filepath.Clean(source.Path)] = string(source.Content)
	}
	diagnostics, err := analyzer.AnalyzeContext(ctx, sources)
	if err != nil {
		return nil, err
	}
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(diagnostic.Code, "syntax.") {
			continue
		}
		documentURI, ok := paths[filepath.Clean(diagnostic.Position.File)]
		if !ok {
			continue
		}
		source := contents[filepath.Clean(diagnostic.Position.File)]
		range_ := sourceRangeToProtocol(source, diagnostic.Range)
		related := make([]protocol.DiagnosticRelatedInformation, 0, len(diagnostic.Related))
		for _, information := range diagnostic.Related {
			relatedURI, exists := paths[filepath.Clean(information.Range.Start.File)]
			if !exists {
				continue
			}
			related = append(related, protocol.DiagnosticRelatedInformation{
				Location: protocol.Location{URI: relatedURI, Range: sourceRangeToProtocol(contents[filepath.Clean(information.Range.Start.File)], information.Range)},
				Message:  information.Message,
			})
		}
		result[documentURI] = append(result[documentURI], protocol.Diagnostic{
			Range: range_, Severity: diagnosticSeverityToProtocol(diagnostic.Severity), RelatedInformation: related,
			Code: protocol.String(diagnostic.Code), Source: protocol.NewOptional("skelc"),
			Message: protocol.String(diagnostic.Message), Data: diagnosticSuggestionData(diagnostic.Suggestion),
		})
	}
	return result, nil
}
