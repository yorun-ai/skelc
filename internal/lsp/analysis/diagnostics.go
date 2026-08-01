// Package analysis runs cancellable semantic analysis for immutable LSP
// workspace snapshots.
package analysis

import (
	"context"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/compiler"
	lspdiagnostic "go.yorun.ai/skelc/internal/lsp/diagnostic"
	"go.yorun.ai/skelc/internal/lsp/index"
)

// SemanticSources converts indexed LSP documents into compiler sources.
func SemanticSources(documents map[uri.URI]*index.Document) ([]compiler.Source, map[string]uri.URI) {
	sources := make([]compiler.Source, 0, len(documents))
	paths := make(map[string]uri.URI, len(documents))
	for documentURI, document := range documents {
		path := filepath.Clean(document.Path)
		sources = append(sources, compiler.Source{
			Path: path, Domain: document.Domain, Root: filepath.Dir(path),
			Content: []byte(document.Source), Parsed: document.Parsed,
			ParseDiagnostics: document.ParseDiagnostics,
		})
		paths[path] = documentURI
	}
	return sources, paths
}

// SemanticDiagnostics analyzes sources and converts compiler diagnostics to
// their LSP representation.
func SemanticDiagnostics(ctx context.Context, workspaceAnalyzer *compiler.WorkspaceAnalyzer, sources []compiler.Source, paths map[string]uri.URI) (map[uri.URI][]protocol.Diagnostic, error) {
	result := map[uri.URI][]protocol.Diagnostic{}
	contents := make(map[string]string, len(sources))
	for _, source := range sources {
		contents[filepath.Clean(source.Path)] = string(source.Content)
	}
	diagnostics, err := workspaceAnalyzer.AnalyzeContext(ctx, sources)
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
		result[documentURI] = append(result[documentURI], lspdiagnostic.ToProtocol(diagnostic, source, func(path string) (uri.URI, string, bool) {
			cleaned := filepath.Clean(path)
			relatedURI, exists := paths[cleaned]
			return relatedURI, contents[cleaned], exists
		}))
	}
	return result, nil
}
