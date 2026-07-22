package lsp

import (
	"strings"
	"unicode"

	"go.lsp.dev/protocol"
)

func indexIncompleteDocument(document *_Document, tokens []_Token) {
	depth := 0
	for index := 0; index < len(tokens); index++ {
		token := tokens[index]
		switch token.Value {
		case "{":
			depth++
			continue
		case "}":
			depth = max(depth-1, 0)
			continue
		}
		if depth != 0 {
			continue
		}
		switch token.Value {
		case "domain":
			name, _ := qualifiedTokenValue(tokens, index+1)
			if name != "" {
				document.Domain = name
			}
		case "import":
			domain, next := qualifiedTokenValue(tokens, index+1)
			if domain == "" {
				continue
			}
			alias := domain[strings.LastIndex(domain, ".")+1:]
			if next+1 < len(tokens) && tokens[next].Value == "as" && isIdentifierValue(tokens[next+1].Value) {
				alias = tokens[next+1].Value
			}
			document.Imports[alias] = domain
		default:
			kind, detail := fallbackEntryKind(token.Value)
			if kind == protocol.SymbolKindNull || index+1 >= len(tokens) || !isIdentifierValue(tokens[index+1].Value) {
				continue
			}
			nameToken := tokens[index+1]
			range_ := offsetRange(document.Source, nameToken.Start, nameToken.End)
			document.Definitions = append(document.Definitions, _Definition{
				Key: document.Domain + "." + nameToken.Value, Name: nameToken.Value, Detail: detail, Kind: kind, Range: range_,
			})
			document.Symbols = append(document.Symbols, _Symbol{Name: nameToken.Value, Detail: detail, Kind: kind, Range: range_})
		}
	}
}

func qualifiedTokenValue(tokens []_Token, start int) (string, int) {
	if start >= len(tokens) || !isIdentifierValue(tokens[start].Value) {
		return "", start
	}
	parts := []string{tokens[start].Value}
	index := start + 1
	for index+1 < len(tokens) && tokens[index].Value == "." && isIdentifierValue(tokens[index+1].Value) {
		parts = append(parts, tokens[index+1].Value)
		index += 2
	}
	return strings.Join(parts, "."), index
}

func fallbackEntryKind(keyword string) (protocol.SymbolKind, string) {
	switch keyword {
	case "enum":
		return protocol.SymbolKindEnum, keyword
	case "data", "config":
		return protocol.SymbolKindStruct, keyword
	case "actor", "service", "web":
		return protocol.SymbolKindInterface, keyword
	case "resource":
		return protocol.SymbolKindObject, keyword
	case "event":
		return protocol.SymbolKindEvent, keyword
	case "task":
		return protocol.SymbolKindFunction, keyword
	default:
		return protocol.SymbolKindNull, ""
	}
}

func isIdentifierValue(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if index == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
		} else if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
