package lsp

import (
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

type _Document struct {
	URI              uri.URI
	Path             string
	Source           string
	Version          int32
	Domain           string
	Imports          map[string]string
	Definitions      []_Definition
	Symbols          []_Symbol
	Occurrences      []_Occurrence
	ParseDiagnostics parser.Diagnostics
	Parsed           *grammar.SkelContent
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
	if path == "" {
		path = documentURI.FsPath()
	}
	document := &_Document{URI: documentURI, Path: path, Source: source, Version: version, Imports: map[string]string{}}
	tokens := scanIdentifiers(source)
	content, diagnostics := parser.ParseSourceRecovering(path, []byte(source))
	document.Parsed = content
	document.ParseDiagnostics = diagnostics
	if len(diagnostics) > 0 {
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
