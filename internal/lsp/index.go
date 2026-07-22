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
	Symbols     []_Symbol
	Occurrences []_Occurrence
	ParseError  error
}

type _Definition struct {
	Key         string
	Name        string
	Detail      string
	Description string
	Kind        protocol.SymbolKind
	Range       protocol.Range
}

type _Occurrence struct {
	Key   string
	Range protocol.Range
}

type _Symbol struct {
	Name        string
	Detail      string
	Description string
	Kind        protocol.SymbolKind
	Range       protocol.Range
	Children    []_Symbol
}

type _Token struct {
	Value string
	Start int
	End   int
}

func indexDocument(documentURI uri.URI, path, source string, version int32) *_Document {
	document := &_Document{URI: documentURI, Source: source, Version: version, Imports: map[string]string{}}
	tokens := scanIdentifiers(source)
	content, err := parser.ParseSource(path, []byte(source))
	if err != nil {
		document.ParseError = err
		indexIncompleteDocument(document, tokens)
		document.Occurrences = indexOccurrences(document, tokens)
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
		name, pos, kind, detail := entryDefinition(entry)
		if name == "" {
			continue
		}
		range_ := identifierRange(source, pos, name)
		description := descriptionFromDecorators(entry.Decorators)
		document.Definitions = append(document.Definitions, _Definition{
			Key: document.Domain + "." + name, Name: name, Detail: detail, Description: description, Kind: kind, Range: range_,
		})
		document.Symbols = append(document.Symbols, entrySymbol(source, entry, name, detail, description, kind, range_))
	}
	document.Occurrences = indexOccurrences(document, tokens)
	return document
}

func entryDefinition(entry *grammar.SkelEntry) (string, lexer.Position, protocol.SymbolKind, string) {
	switch {
	case entry.Enum != nil:
		return entry.Enum.Name.Value, entry.Enum.Name.Pos, protocol.SymbolKindEnum, "enum"
	case entry.Data != nil:
		return entry.Data.Name.Value, entry.Data.Name.Pos, protocol.SymbolKindStruct, "data"
	case entry.Config != nil:
		return entry.Config.Name.Value, entry.Config.Name.Pos, protocol.SymbolKindStruct, "config"
	case entry.Actor != nil:
		return entry.Actor.Name.Value, entry.Actor.Name.Pos, protocol.SymbolKindInterface, "actor"
	case entry.Resource != nil:
		return entry.Resource.Name.Value, entry.Resource.Name.Pos, protocol.SymbolKindObject, "resource"
	case entry.Service != nil:
		return entry.Service.Name.Value, entry.Service.Name.Pos, protocol.SymbolKindInterface, "service"
	case entry.Web != nil:
		return entry.Web.Name.Value, entry.Web.Name.Pos, protocol.SymbolKindInterface, "web"
	case entry.Event != nil:
		return entry.Event.Name.Value, entry.Event.Name.Pos, protocol.SymbolKindEvent, "event"
	case entry.Task != nil:
		return entry.Task.Name.Value, entry.Task.Name.Pos, protocol.SymbolKindFunction, "task"
	default:
		return "", lexer.Position{}, protocol.SymbolKindNull, ""
	}
}

func entrySymbol(source string, entry *grammar.SkelEntry, name, detail, description string, kind protocol.SymbolKind, range_ protocol.Range) _Symbol {
	children := []_Symbol{}
	switch {
	case entry.Enum != nil:
		for _, item := range entry.Enum.Items {
			children = append(children, newSymbol(source, item.Name, "enum item", descriptionFromDecorators(item.Decorators), protocol.SymbolKindEnumMember, nil))
		}
	case entry.Data != nil:
		children = dataMemberSymbols(source, entry.Data.Members)
	case entry.Config != nil:
		children = dataMemberSymbols(source, entry.Config.Members)
	case entry.Actor != nil:
		for _, via := range entry.Actor.Vias {
			children = append(children, newSymbol(source, via.Name, "actor transport", "", protocol.SymbolKindInterface, nil))
		}
		for _, section := range entry.Actor.Sections {
			if section.Auth == nil {
				continue
			}
			if section.Auth.Credential != nil {
				children = append(children, sectionSymbol(source, "credential", section.Auth.Credential.Pos, dataMemberSymbols(source, section.Auth.Credential.Members)))
			}
			if section.Auth.Info != nil {
				children = append(children, sectionSymbol(source, "info", section.Auth.Info.Pos, dataMemberSymbols(source, section.Auth.Info.Members)))
			}
		}
	case entry.Resource != nil:
		for _, section := range entry.Resource.Sections {
			if section.Check != nil {
				children = append(children, resourceCheckSymbol(source, section.Check))
			}
			if section.Action != nil {
				actionChildren := make([]_Symbol, 0, len(section.Action.Checks))
				for _, check := range section.Action.Checks {
					actionChildren = append(actionChildren, resourceCheckSymbol(source, check))
				}
				children = append(children, newSymbol(source, section.Action.Name, "action", descriptionFromDecorators(section.Action.Decorators), protocol.SymbolKindMethod, actionChildren))
			}
		}
	case entry.Service != nil:
		for _, section := range entry.Service.Sections {
			if section.Method == nil {
				continue
			}
			method := section.Method
			methodChildren := []_Symbol{}
			if method.Input != nil {
				methodChildren = argumentSymbols(source, method.Input.Arguments)
			}
			children = append(children, newSymbol(source, method.Name, "method", descriptionFromDecoratorGroups(section.Decorators, method.Decorators), protocol.SymbolKindMethod, methodChildren))
		}
	case entry.Event != nil:
		if entry.Event.Payload != nil {
			children = dataMemberSymbols(source, entry.Event.Payload.Members)
		}
	case entry.Task != nil:
		for _, trigger := range entry.Task.Triggers {
			triggerChildren := []_Symbol{}
			if trigger.Input != nil {
				triggerChildren = argumentSymbols(source, trigger.Input.Arguments)
			}
			children = append(children, newSymbol(source, trigger.Name, "trigger", descriptionFromDecorators(trigger.Decorators), protocol.SymbolKindEvent, triggerChildren))
		}
	}
	return finishSymbol(_Symbol{Name: name, Detail: detail, Description: description, Kind: kind, Range: range_, Children: children})
}

func dataMemberSymbols(source string, members []*grammar.DataMember) []_Symbol {
	symbols := make([]_Symbol, 0, len(members))
	for _, member := range members {
		symbols = append(symbols, newSymbol(source, member.Name, "field", descriptionFromDecorators(member.Decorators), protocol.SymbolKindField, nil))
	}
	return symbols
}

func argumentSymbols(source string, arguments []*grammar.Argument) []_Symbol {
	symbols := make([]_Symbol, 0, len(arguments))
	for _, argument := range arguments {
		symbols = append(symbols, newSymbol(source, argument.Name, "parameter", descriptionFromDecorators(argument.Decorators), protocol.SymbolKindVariable, nil))
	}
	return symbols
}

func resourceCheckSymbol(source string, check *grammar.ResourceCheck) _Symbol {
	children := argumentSymbols(source, check.Arguments)
	return newSymbol(source, check.Name, "check", "", protocol.SymbolKindFunction, children)
}

func newSymbol(source string, name *grammar.Identifier, detail, description string, kind protocol.SymbolKind, children []_Symbol) _Symbol {
	range_ := identifierRange(source, name.Pos, name.Value)
	return finishSymbol(_Symbol{Name: name.Value, Detail: detail, Description: description, Kind: kind, Range: range_, Children: children})
}

func sectionSymbol(source, name string, pos lexer.Position, children []_Symbol) _Symbol {
	range_ := identifierRange(source, pos, name)
	return finishSymbol(_Symbol{Name: name, Detail: "actor auth section", Kind: protocol.SymbolKindObject, Range: range_, Children: children})
}

func finishSymbol(symbol _Symbol) _Symbol {
	for _, child := range symbol.Children {
		if comparePosition(child.Range.End, symbol.Range.End) > 0 {
			symbol.Range.End = child.Range.End
		}
	}
	return symbol
}

func descriptionFromDecorators(decorators []*grammar.Decorator) string {
	for _, decorator := range decorators {
		if decorator.Name.Value == "desc" && decorator.Value != nil {
			return grammar.UnquoteDescriptionString(decorator.Value.Raw)
		}
	}
	return ""
}

func descriptionFromDecoratorGroups(groups ...[]*grammar.Decorator) string {
	for _, decorators := range groups {
		if description := descriptionFromDecorators(decorators); description != "" {
			return description
		}
	}
	return ""
}

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
