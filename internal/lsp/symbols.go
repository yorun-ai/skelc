package lsp

import (
	"context"
	"slices"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (s *_Server) Symbols(_ context.Context, params *protocol.WorkspaceSymbolParams) (protocol.WorkspaceSymbolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	query := strings.ToLower(strings.TrimSpace(params.Query))
	result := protocol.SymbolInformationSlice{}
	for _, document := range s.documents {
		appendWorkspaceSymbols(&result, document.URI, document.Symbols, "", query)
	}
	slices.SortFunc(result, func(left, right protocol.SymbolInformation) int {
		if compared := strings.Compare(strings.ToLower(left.Name), strings.ToLower(right.Name)); compared != 0 {
			return compared
		}
		if compared := strings.Compare(string(left.Location.URI), string(right.Location.URI)); compared != 0 {
			return compared
		}
		return comparePosition(left.Location.Range.Start, right.Location.Range.Start)
	})
	return result, nil
}

func documentSymbols(symbols []_Symbol) protocol.DocumentSymbolSlice {
	result := make(protocol.DocumentSymbolSlice, 0, len(symbols))
	for _, symbol := range symbols {
		detail := symbol.Detail
		result = append(result, protocol.DocumentSymbol{
			Name: symbol.Name, Detail: &detail, Kind: symbol.Kind, Range: symbol.Range,
			SelectionRange: selectionRange(symbol), Children: documentSymbols(symbol.Children),
		})
	}
	return result
}

func selectionRange(symbol _Symbol) protocol.Range {
	range_ := symbol.Range
	range_.End = protocol.Position{Line: range_.Start.Line, Character: range_.Start.Character + uint32(utf16Length(symbol.Name))}
	return range_
}

func appendWorkspaceSymbols(result *protocol.SymbolInformationSlice, documentURI uri.URI, symbols []_Symbol, container, query string) {
	for _, symbol := range symbols {
		if query == "" || strings.Contains(strings.ToLower(symbol.Name), query) {
			information := protocol.SymbolInformation{
				BaseSymbolInformation: protocol.BaseSymbolInformation{Name: symbol.Name, Kind: symbol.Kind},
				Location:              protocol.Location{URI: documentURI, Range: selectionRange(symbol)},
			}
			if container != "" {
				information.ContainerName = new(container)
			}
			*result = append(*result, information)
		}
		nextContainer := symbol.Name
		if container != "" {
			nextContainer = container + "." + symbol.Name
		}
		appendWorkspaceSymbols(result, documentURI, symbol.Children, nextContainer, query)
	}
}
