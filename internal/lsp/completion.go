package lsp

import (
	"context"
	"slices"

	"go.lsp.dev/protocol"
)

var completionKeywords = []string{
	"actor", "action", "all", "any", "as", "auth", "check", "config", "credential", "data",
	"domain", "enum", "event", "for", "import", "info", "input", "method", "noauth", "output",
	"payload", "permission", "pub", "require", "resource", "service", "task", "trigger", "via", "web",
}

var completionTypes = []string{
	"binary", "bool", "decimal", "duration", "float", "int", "json", "list", "localdate",
	"localdatetime", "localtime", "map", "PermissionCode", "string", "timestamp", "uuid",
}

func (s *_Server) Completion(_ context.Context, params *protocol.CompletionParams) (protocol.CompletionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return protocol.CompletionItemSlice{}, nil
	}
	if positionInNonCode(document.Source, params.Position) {
		return protocol.CompletionItemSlice{}, nil
	}

	items := map[string]protocol.CompletionItem{}
	qualifier := qualifierBeforePosition(document.Source, params.Position)
	if qualifier != "" {
		domain := document.Imports[qualifier]
		for _, candidate := range s.documents {
			if candidate.Domain != domain {
				continue
			}
			for _, definition := range candidate.Definitions {
				items[definition.Name] = symbolCompletion(definition, domain)
			}
		}
	} else {
		for _, keyword := range completionKeywords {
			items[keyword] = protocol.CompletionItem{
				Label: keyword, Kind: protocol.CompletionItemKindKeyword,
				Detail: protocol.NewOptional("Skel keyword"),
			}
		}
		for _, kind := range completionTypes {
			items[kind] = protocol.CompletionItem{
				Label: kind, Kind: protocol.CompletionItemKindClass,
				Detail: protocol.NewOptional("Skel built-in type"),
			}
		}
		for alias, domain := range document.Imports {
			items[alias] = protocol.CompletionItem{
				Label: alias, Kind: protocol.CompletionItemKindModule,
				Detail: protocol.NewOptional(domain),
			}
		}
		for _, candidate := range s.documents {
			if candidate.Domain != document.Domain {
				continue
			}
			for _, definition := range candidate.Definitions {
				items[definition.Name] = symbolCompletion(definition, document.Domain)
			}
		}
	}

	labels := make([]string, 0, len(items))
	for label := range items {
		labels = append(labels, label)
	}
	slices.Sort(labels)
	result := make(protocol.CompletionItemSlice, 0, len(labels))
	for _, label := range labels {
		result = append(result, items[label])
	}
	return result, nil
}

func symbolCompletion(definition _Definition, domain string) protocol.CompletionItem {
	item := protocol.CompletionItem{
		Label: definition.Name, Kind: completionKind(definition.Kind), Detail: protocol.NewOptional(domain + "." + definition.Name),
	}
	if definition.Description != "" {
		item.Documentation = protocol.String(definition.Description)
	}
	return item
}

func completionKind(kind protocol.SymbolKind) protocol.CompletionItemKind {
	switch kind {
	case protocol.SymbolKindEnum:
		return protocol.CompletionItemKindEnum
	case protocol.SymbolKindInterface:
		return protocol.CompletionItemKindInterface
	case protocol.SymbolKindEvent:
		return protocol.CompletionItemKindEvent
	case protocol.SymbolKindFunction:
		return protocol.CompletionItemKindFunction
	case protocol.SymbolKindStruct, protocol.SymbolKindObject:
		return protocol.CompletionItemKindStruct
	default:
		return protocol.CompletionItemKindReference
	}
}
