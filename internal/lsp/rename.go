package lsp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (s *_Server) PrepareRename(_ context.Context, params *protocol.PrepareRenameParams) (protocol.PrepareRenameResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return nil, nil
	}
	occurrence, ok := occurrenceAt(document, params.Position)
	if !ok || !s.hasDefinitionLocked(occurrence.Key) {
		return nil, nil
	}
	range_ := occurrence.Range
	return &range_, nil
}

func (s *_Server) Rename(_ context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	if !isIdentifierValue(params.NewName) {
		return nil, fmt.Errorf("invalid Skel identifier %q", params.NewName)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return nil, nil
	}
	occurrence, ok := occurrenceAt(document, params.Position)
	if !ok || !s.hasDefinitionLocked(occurrence.Key) {
		return nil, nil
	}
	for _, candidate := range s.documents {
		if candidate.Domain != domainFromKey(occurrence.Key) {
			continue
		}
		for _, definition := range candidate.Definitions {
			if definition.Name == params.NewName && definition.Key != occurrence.Key {
				return nil, fmt.Errorf("Skel declaration %s already exists", definition.Key)
			}
		}
	}
	changes := map[uri.URI][]protocol.TextEdit{}
	for _, candidate := range s.documents {
		for _, reference := range candidate.Occurrences {
			if reference.Key == occurrence.Key {
				changes[candidate.URI] = append(changes[candidate.URI], protocol.TextEdit{Range: reference.Range, NewText: params.NewName})
			}
		}
	}
	for documentURI := range changes {
		slices.SortFunc(changes[documentURI], func(left, right protocol.TextEdit) int {
			return comparePosition(left.Range.Start, right.Range.Start)
		})
	}
	return &protocol.WorkspaceEdit{Changes: changes}, nil
}

func (s *_Server) hasDefinitionLocked(key string) bool {
	for _, document := range s.documents {
		for _, definition := range document.Definitions {
			if definition.Key == key {
				return true
			}
		}
	}
	return false
}

func domainFromKey(key string) string {
	if index := strings.LastIndex(key, "."); index >= 0 {
		return key[:index]
	}
	return ""
}
