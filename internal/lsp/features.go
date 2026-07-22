package lsp

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/formatter"
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

func (s *_Server) Formatting(_ context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	s.mu.RLock()
	document := s.documents[params.TextDocument.URI]
	s.mu.RUnlock()
	if document == nil || document.ParseError != nil {
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

func qualifierBeforePosition(source string, position protocol.Position) string {
	offset := positionOffset(source, position)
	start := offset
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(source[:start])
		if r != '_' && !isLetterOrDigit(r) {
			break
		}
		start -= size
	}
	if start == 0 || source[start-1] != '.' {
		return ""
	}
	end := start - 1
	start = end
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(source[:start])
		if r != '_' && !isLetterOrDigit(r) {
			break
		}
		start -= size
	}
	return source[start:end]
}

func positionOffset(source string, position protocol.Position) int {
	lineStart := 0
	for line := uint32(0); line < position.Line && lineStart < len(source); line++ {
		next := strings.IndexByte(source[lineStart:], '\n')
		if next < 0 {
			return len(source)
		}
		lineStart += next + 1
	}
	offset := lineStart
	units := uint32(0)
	for offset < len(source) && source[offset] != '\n' && units < position.Character {
		r, size := utf8.DecodeRuneInString(source[offset:])
		width := uint32(1)
		if r > 0xffff {
			width = 2
		}
		if units+width > position.Character {
			break
		}
		units += width
		offset += size
	}
	return offset
}

func positionInNonCode(source string, position protocol.Position) bool {
	offset := positionOffset(source, position)
	const (
		stateCode = iota
		stateLineComment
		stateBlockComment
		stateString
		stateTripleString
	)
	state := stateCode
	for index := 0; index < offset; {
		switch state {
		case stateCode:
			switch {
			case strings.HasPrefix(source[index:], "//"):
				state = stateLineComment
				index += 2
			case strings.HasPrefix(source[index:], "/*"):
				state = stateBlockComment
				index += 2
			case strings.HasPrefix(source[index:], `"""`):
				state = stateTripleString
				index += 3
			case source[index] == '"':
				state = stateString
				index++
			default:
				index++
			}
		case stateLineComment:
			if source[index] == '\n' {
				state = stateCode
			}
			index++
		case stateBlockComment:
			if strings.HasPrefix(source[index:], "*/") {
				state = stateCode
				index += 2
			} else {
				index++
			}
		case stateString:
			if source[index] == '\\' {
				index += min(2, offset-index)
			} else {
				if source[index] == '"' {
					state = stateCode
				}
				index++
			}
		case stateTripleString:
			if strings.HasPrefix(source[index:], `"""`) {
				state = stateCode
				index += 3
			} else {
				index++
			}
		}
	}
	return state != stateCode
}

func isLetterOrDigit(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
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
