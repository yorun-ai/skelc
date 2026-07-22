package lsp

import (
	"context"
	"errors"

	"github.com/alecthomas/participle/v2"
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
	if document != nil && document.ParseError != nil {
		position := protocol.Position{}
		message := document.ParseError.Error()
		var parseError participle.Error
		if errors.As(document.ParseError, &parseError) {
			parsePosition := parseError.Position()
			position = identifierRange(document.Source, parsePosition, "").Start
			message = parseError.Message()
		}
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{Start: position, End: protocol.Position{Line: position.Line, Character: position.Character + 1}}, Severity: protocol.DiagnosticSeverityError,
			Source: protocol.NewOptional("skelc"), Message: protocol.String(message),
		})
	}
	params := &protocol.PublishDiagnosticsParams{URI: documentURI, Diagnostics: diagnostics}
	if document != nil {
		params.Version = protocol.NewOptional(document.Version)
	}
	return client.PublishDiagnostics(ctx, params)
}
