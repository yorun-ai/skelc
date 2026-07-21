package lsp

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/alecthomas/participle/v2/lexer"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

type _Document struct {
	URI         uri.URI
	Source      string
	Version     int32
	Domain      string
	Imports     map[string]string
	Definitions []_Definition
	Occurrences []_Occurrence
	ParseError  error
}

type _Definition struct {
	Key   string
	Name  string
	Kind  protocol.SymbolKind
	Range protocol.Range
}

type _Occurrence struct {
	Key   string
	Range protocol.Range
}

type _Token struct {
	Value string
	Start int
	End   int
}

func indexDocument(documentURI uri.URI, path, source string, version int32) *_Document {
	document := &_Document{URI: documentURI, Source: source, Version: version, Imports: map[string]string{}}
	content, err := parser.ParseSource(path, []byte(source))
	if err != nil {
		document.ParseError = err
		return document
	}
	if content.Domain != nil && content.Domain.Name != nil {
		document.Domain = content.Domain.Name.String()
	}
	for _, importDecl := range content.Imports {
		domain := importDecl.Domain.String()
		alias := domain[strings.LastIndex(domain, ".")+1:]
		if importDecl.Alias != nil {
			alias = importDecl.Alias.Value
		}
		document.Imports[alias] = domain
	}
	for _, entry := range content.Entries {
		name, pos, kind := entryDefinition(entry)
		if name == "" {
			continue
		}
		document.Definitions = append(document.Definitions, _Definition{
			Key: document.Domain + "." + name, Name: name, Kind: kind, Range: identifierRange(source, pos, name),
		})
	}
	document.Occurrences = indexOccurrences(document, scanIdentifiers(source))
	return document
}

func entryDefinition(entry *grammar.SkelEntry) (string, lexer.Position, protocol.SymbolKind) {
	switch {
	case entry.Enum != nil:
		return entry.Enum.Name.Value, entry.Enum.Name.Pos, protocol.SymbolKindEnum
	case entry.Data != nil:
		return entry.Data.Name.Value, entry.Data.Name.Pos, protocol.SymbolKindStruct
	case entry.Config != nil:
		return entry.Config.Name.Value, entry.Config.Name.Pos, protocol.SymbolKindStruct
	case entry.Actor != nil:
		return entry.Actor.Name.Value, entry.Actor.Name.Pos, protocol.SymbolKindInterface
	case entry.Resource != nil:
		return entry.Resource.Name.Value, entry.Resource.Name.Pos, protocol.SymbolKindObject
	case entry.Service != nil:
		return entry.Service.Name.Value, entry.Service.Name.Pos, protocol.SymbolKindInterface
	case entry.Web != nil:
		return entry.Web.Name.Value, entry.Web.Name.Pos, protocol.SymbolKindInterface
	case entry.Event != nil:
		return entry.Event.Name.Value, entry.Event.Name.Pos, protocol.SymbolKindEvent
	case entry.Task != nil:
		return entry.Task.Name.Value, entry.Task.Name.Pos, protocol.SymbolKindFunction
	default:
		return "", lexer.Position{}, protocol.SymbolKindNull
	}
}

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
		case source[offset] == '.':
			tokens = append(tokens, _Token{Value: ".", Start: offset, End: offset + 1})
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

func occurrenceAt(document *_Document, position protocol.Position) (_Occurrence, bool) {
	for _, occurrence := range document.Occurrences {
		if containsPosition(occurrence.Range, position) {
			return occurrence, true
		}
	}
	return _Occurrence{}, false
}

func containsPosition(r protocol.Range, position protocol.Position) bool {
	return comparePosition(r.Start, position) <= 0 && comparePosition(position, r.End) < 0
}

func comparePosition(left, right protocol.Position) int {
	if left.Line != right.Line {
		return int(left.Line) - int(right.Line)
	}
	return int(left.Character) - int(right.Character)
}

func identifierRange(source string, pos lexer.Position, name string) protocol.Range {
	line := max(pos.Line-1, 0)
	byteColumn := max(pos.Column-1, 0)
	lineSource := sourceLine(source, line)
	byteColumn = min(byteColumn, len(lineSource))
	start := protocol.Position{Line: uint32(line), Character: uint32(utf16Length(lineSource[:byteColumn]))}
	return protocol.Range{Start: start, End: protocol.Position{Line: start.Line, Character: start.Character + uint32(utf16Length(name))}}
}

func offsetRange(source string, start, end int) protocol.Range {
	return protocol.Range{Start: offsetPosition(source, start), End: offsetPosition(source, end)}
}

func offsetPosition(source string, offset int) protocol.Position {
	offset = min(max(offset, 0), len(source))
	line := strings.Count(source[:offset], "\n")
	lineStart := strings.LastIndexByte(source[:offset], '\n') + 1
	return protocol.Position{Line: uint32(line), Character: uint32(utf16Length(source[lineStart:offset]))}
}

func sourceLine(source string, line int) string {
	lines := strings.Split(source, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return strings.TrimSuffix(lines[line], "\r")
}

func utf16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func firstRune(value string) rune {
	r, _ := utf8.DecodeRuneInString(value)
	return r
}

func sortLocations(locations []protocol.Location) {
	slices.SortFunc(locations, func(left, right protocol.Location) int {
		if compared := strings.Compare(string(left.URI), string(right.URI)); compared != 0 {
			return compared
		}
		return comparePosition(left.Range.Start, right.Range.Start)
	})
}
