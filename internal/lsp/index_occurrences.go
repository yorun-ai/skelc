package lsp

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func indexOccurrences(document *_Document, tokens []_Token) []_Occurrence {
	definitions := make(map[string]string, len(document.Definitions))
	for _, definition := range document.Definitions {
		definitions[definition.Name] = definition.Key
	}
	occurrences := make([]_Occurrence, 0)
	for index, token := range tokens {
		key := ""
		if index >= 2 && tokens[index-1].Value == "." {
			if domain := document.Imports[tokens[index-2].Value]; domain != "" {
				key = domain + "." + token.Value
			}
		} else if definitions[token.Value] != "" {
			key = definitions[token.Value]
		} else if unicode.IsUpper(firstRune(token.Value)) && document.Domain != "" {
			key = document.Domain + "." + token.Value
		}
		if key != "" {
			occurrences = append(occurrences, _Occurrence{Key: key, Range: offsetRange(document.Source, token.Start, token.End)})
		}
	}
	return occurrences
}

func scanIdentifiers(source string) []_Token {
	tokens := make([]_Token, 0)
	for offset := 0; offset < len(source); {
		switch {
		case strings.HasPrefix(source[offset:], "//"):
			if end := strings.IndexByte(source[offset:], '\n'); end >= 0 {
				offset += end
			} else {
				return tokens
			}
		case strings.HasPrefix(source[offset:], "/*"):
			if end := strings.Index(source[offset+2:], "*/"); end >= 0 {
				offset += end + 4
			} else {
				return tokens
			}
		case strings.HasPrefix(source[offset:], `"""`):
			if end := strings.Index(source[offset+3:], `"""`); end >= 0 {
				offset += end + 6
			} else {
				return tokens
			}
		case source[offset] == '"':
			offset++
			for offset < len(source) {
				if source[offset] == '\\' {
					offset += min(2, len(source)-offset)
					continue
				}
				offset++
				if source[offset-1] == '"' {
					break
				}
			}
		case source[offset] == '.' || source[offset] == '{' || source[offset] == '}':
			tokens = append(tokens, _Token{Value: source[offset : offset+1], Start: offset, End: offset + 1})
			offset++
		default:
			r, size := utf8.DecodeRuneInString(source[offset:])
			if r == '_' || unicode.IsLetter(r) {
				start := offset
				offset += size
				for offset < len(source) {
					r, size = utf8.DecodeRuneInString(source[offset:])
					if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
						break
					}
					offset += size
				}
				tokens = append(tokens, _Token{Value: source[start:offset], Start: start, End: offset})
			} else {
				offset += size
			}
		}
	}
	return tokens
}
