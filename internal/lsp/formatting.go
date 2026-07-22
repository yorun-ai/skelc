package lsp

import (
	"context"

	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/formatter"
)

func (s *_Server) Formatting(_ context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	s.mu.RLock()
	document := s.documents[params.TextDocument.URI]
	s.mu.RUnlock()
	if document == nil || len(document.ParseDiagnostics) > 0 {
		return []protocol.TextEdit{}, nil
	}
	formatted := string(formatter.Source([]byte(document.Source)))
	if formatted == document.Source {
		return []protocol.TextEdit{}, nil
	}
	return []protocol.TextEdit{{
		Range:   protocol.Range{Start: protocol.Position{}, End: offsetPosition(document.Source, len(document.Source))},
		NewText: formatted,
	}}, nil
}
