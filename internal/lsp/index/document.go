// Package index builds immutable language indexes for Skel documents.
package index

import (
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/lsp/source"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

type Document struct {
	URI              uri.URI
	Path             string
	Source           string
	Version          int32
	Domain           string
	Imports          map[string]string
	Definitions      []Definition
	Symbols          []Symbol
	Occurrences      []Occurrence
	ParseDiagnostics parser.Diagnostics
	Parsed           *grammar.SkelContent
}

type Definition struct {
	Key         string
	Name        string
	Detail      string
	Description string
	Deprecated  bool
	Kind        protocol.SymbolKind
	Range       protocol.Range
}

type Occurrence struct {
	Key   string
	Range protocol.Range
}

type Symbol struct {
	Name        string
	Detail      string
	Description string
	Deprecated  bool
	Kind        protocol.SymbolKind
	Range       protocol.Range
	Children    []Symbol
}

// Build parses and indexes one in-memory Skel document.
func Build(documentURI uri.URI, path, content string, version int32) *Document {
	if path == "" {
		path = documentURI.FsPath()
	}
	document := &Document{URI: documentURI, Path: path, Source: content, Version: version, Imports: map[string]string{}}
	tokens := source.New(content).IdentifierTokens()
	parsed, diagnostics := parser.ParseSourceRecovering(path, []byte(content))
	document.Parsed = parsed
	document.ParseDiagnostics = diagnostics
	if len(diagnostics) > 0 {
		indexIncompleteDocument(document, tokens)
		document.Occurrences = indexOccurrences(document, tokens)
		return document
	}
	if parsed.Domain != nil && parsed.Domain.Name != nil {
		document.Domain = parsed.Domain.Name.String()
	}
	for _, importDecl := range parsed.Imports {
		domain := importDecl.Domain.String()
		alias := domain[strings.LastIndex(domain, ".")+1:]
		if importDecl.Alias != nil {
			alias = importDecl.Alias.Value
		}
		document.Imports[alias] = domain
	}
	for _, entry := range parsed.Entries {
		name, pos, kind, detail := entryDefinition(entry)
		if name == "" {
			continue
		}
		range_ := identifierRange(content, pos, name)
		description, deprecated := documentationFromDecoratorGroups(entry.Decorators)
		document.Definitions = append(document.Definitions, Definition{
			Key: document.Domain + "." + name, Name: name, Detail: detail, Description: description,
			Deprecated: deprecated, Kind: kind, Range: range_,
		})
		document.Symbols = append(document.Symbols, entrySymbol(content, entry, name, detail, description, deprecated, kind, range_))
	}
	document.Occurrences = indexOccurrences(document, tokens)
	return document
}
