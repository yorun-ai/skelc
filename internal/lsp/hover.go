package lsp

import (
	"context"
	"slices"

	"go.lsp.dev/protocol"
)

func (s *_Server) Hover(_ context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return nil, nil
	}
	if occurrence, ok := occurrenceAt(document, params.Position); ok {
		for _, candidate := range s.documents {
			for _, definition := range candidate.Definitions {
				if definition.Key == occurrence.Key {
					return hoverResult(occurrence.Range, definition.Detail, definition.Key, definition.Description), nil
				}
			}
		}
	}
	if symbol, ok := symbolAt(document.Symbols, params.Position); ok {
		return hoverResult(selectionRange(symbol), symbol.Detail, symbol.Name, symbol.Description), nil
	}
	for _, token := range scanIdentifiers(document.Source) {
		range_ := offsetRange(document.Source, token.Start, token.End)
		if containsPosition(range_, params.Position) && slices.Contains(completionTypes, token.Value) {
			return hoverResult(range_, "built-in type", token.Value, ""), nil
		}
	}
	return nil, nil
}

func hoverResult(range_ protocol.Range, detail, name, description string) *protocol.Hover {
	if detail == "" {
		detail = "declaration"
	}
	value := "```skel\n" + detail + " " + name + "\n```"
	if description != "" {
		value += "\n\n" + description
	}
	return &protocol.Hover{Contents: &protocol.MarkupContent{Kind: protocol.MarkupKindMarkdown, Value: value}, Range: &range_}
}

func symbolAt(symbols []_Symbol, position protocol.Position) (_Symbol, bool) {
	for _, symbol := range symbols {
		selection := selectionRange(symbol)
		if containsPosition(selection, position) {
			return symbol, true
		}
		if child, ok := symbolAt(symbol.Children, position); ok {
			return child, true
		}
	}
	return _Symbol{}, false
}
