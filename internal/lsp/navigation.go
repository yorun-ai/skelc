package lsp

import (
	"context"

	"go.lsp.dev/protocol"
)

func (s *_Server) Definition(_ context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return protocol.LocationSlice{}, nil
	}
	occurrence, ok := occurrenceAt(document, params.Position)
	if !ok {
		return protocol.LocationSlice{}, nil
	}
	locations := make([]protocol.Location, 0)
	for _, candidate := range s.documents {
		for _, definition := range candidate.Definitions {
			if definition.Key == occurrence.Key {
				locations = append(locations, protocol.Location{URI: candidate.URI, Range: definition.Range})
			}
		}
	}
	sortLocations(locations)
	return protocol.LocationSlice(locations), nil
}

func (s *_Server) References(_ context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return []protocol.Location{}, nil
	}
	occurrence, ok := occurrenceAt(document, params.Position)
	if !ok {
		return []protocol.Location{}, nil
	}
	locations := make([]protocol.Location, 0)
	definitions := make(map[protocol.Location]bool)
	for _, candidate := range s.documents {
		for _, definition := range candidate.Definitions {
			if definition.Key == occurrence.Key {
				definitions[protocol.Location{URI: candidate.URI, Range: definition.Range}] = true
			}
		}
	}
	for _, candidate := range s.documents {
		for _, reference := range candidate.Occurrences {
			if reference.Key == occurrence.Key {
				location := protocol.Location{URI: candidate.URI, Range: reference.Range}
				if params.Context.IncludeDeclaration || !definitions[location] {
					locations = append(locations, location)
				}
			}
		}
	}
	sortLocations(locations)
	return locations, nil
}

func (s *_Server) DocumentSymbol(_ context.Context, params *protocol.DocumentSymbolParams) (protocol.DocumentSymbolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return protocol.DocumentSymbolSlice{}, nil
	}
	return documentSymbols(document.Symbols), nil
}
