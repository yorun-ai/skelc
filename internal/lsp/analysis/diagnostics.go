// Package analysis runs cancellable semantic analysis for immutable LSP
// workspace snapshots.
package analysis

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/compiler"
	"go.yorun.ai/skelc/internal/lsp/index"
	"go.yorun.ai/skelc/internal/lsp/source"
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

func sourceRangeToProtocol(content string, sourceRange compiler.SourceRange) protocol.Range {
	buffer := source.New(content)
	start := buffer.IdentifierRange(sourceRange.Start.Line, sourceRange.Start.Column, "").Start
	end := buffer.IdentifierRange(sourceRange.End.Line, sourceRange.End.Column, "").Start
	if comparePosition(end, start) <= 0 {
		end = start
		end.Character++
	}
	return protocol.Range{Start: start, End: end}
}

func diagnosticSeverityToProtocol(severity compiler.DiagnosticSeverity) protocol.DiagnosticSeverity {
	if severity == compiler.DiagnosticSeverityWarning {
		return protocol.DiagnosticSeverityWarning
	}
	return protocol.DiagnosticSeverityError
}

func diagnosticSuggestionData(suggestion *compiler.DiagnosticSuggestion) protocol.LSPAny {
	if suggestion == nil {
		return nil
	}
	content, err := json.Marshal(suggestion)
	if err != nil {
		return nil
	}
	return protocol.LSPAny(content)
}

func comparePosition(left, right protocol.Position) int {
	if left.Line != right.Line {
		return int(left.Line) - int(right.Line)
	}
	return int(left.Character) - int(right.Character)
}
