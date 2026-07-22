package lsp

import (
	"context"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (s *_Server) publishDiagnostics(ctx context.Context, documentURI uri.URI) error {
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return nil
	}
	s.rememberClient(ctx)
	return s.publishDiagnosticsWithClient(ctx, client, documentURI)
}

func (s *_Server) publishDiagnosticsWithClient(ctx context.Context, client protocol.Client, documentURI uri.URI) error {
	s.mu.RLock()
	document := s.documents[documentURI]
	semantic := append([]protocol.Diagnostic{}, s.semantic[documentURI]...)
	s.mu.RUnlock()
	diagnostics := semantic
	if document != nil {
		for _, diagnostic := range document.ParseDiagnostics {
			diagnostics = append(diagnostics, protocol.Diagnostic{
				Range: sourceRangeToProtocol(document.Source, diagnostic.Range), Severity: diagnosticSeverityToProtocol(diagnostic.Severity),
				Code: protocol.String(diagnostic.Code), Source: protocol.NewOptional("skelc"),
				Message: protocol.String(diagnostic.Message), Data: diagnosticSuggestionData(diagnostic.Suggestion),
			})
		}
	}
	params := &protocol.PublishDiagnosticsParams{URI: documentURI, Diagnostics: diagnostics}
	if document != nil {
		params.Version = protocol.NewOptional(document.Version)
	}
	return client.PublishDiagnostics(ctx, params)
}
