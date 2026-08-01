package workspace

import (
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/lsp/index"
)

// Snapshot is an immutable workspace view used by one request or analysis.
type Snapshot struct {
	revision    uint64
	documents   map[uri.URI]*index.Document
	ordered     []*index.Document
	byDomain    map[string][]*index.Document
	definitions map[string][]DefinitionLocation
	occurrences map[string][]OccurrenceLocation
}

// DefinitionLocation identifies a definition and its containing document.
type DefinitionLocation struct {
	Document   *index.Document
	Definition index.Definition
}

// OccurrenceLocation identifies an occurrence and its containing document.
type OccurrenceLocation struct {
	Document   *index.Document
	Occurrence index.Occurrence
}

func newSnapshot(revision uint64, documents map[uri.URI]*index.Document, ordered []*index.Document) Snapshot {
	snapshot := Snapshot{
		revision: revision, documents: documents, ordered: ordered,
		byDomain: map[string][]*index.Document{}, definitions: map[string][]DefinitionLocation{},
		occurrences: map[string][]OccurrenceLocation{},
	}
	for _, document := range ordered {
		snapshot.byDomain[document.Domain] = append(snapshot.byDomain[document.Domain], document)
		for _, definition := range document.Definitions {
			snapshot.definitions[definition.Key] = append(snapshot.definitions[definition.Key], DefinitionLocation{Document: document, Definition: definition})
		}
		for _, occurrence := range document.Occurrences {
			snapshot.occurrences[occurrence.Key] = append(snapshot.occurrences[occurrence.Key], OccurrenceLocation{Document: document, Occurrence: occurrence})
		}
	}
	return snapshot
}

// Revision identifies the store state represented by the snapshot.
func (s Snapshot) Revision() uint64 { return s.revision }

// Document returns a document by URI.
func (s Snapshot) Document(documentURI uri.URI) *index.Document { return s.documents[documentURI] }

// Documents returns all documents in stable URI order.
func (s Snapshot) Documents() []*index.Document { return append([]*index.Document{}, s.ordered...) }

// DocumentsMap returns a copy of the URI lookup map.
func (s Snapshot) DocumentsMap() map[uri.URI]*index.Document {
	result := make(map[uri.URI]*index.Document, len(s.documents))
	for documentURI, document := range s.documents {
		result[documentURI] = document
	}
	return result
}

// DocumentsInDomain returns documents belonging to domain in stable order.
func (s Snapshot) DocumentsInDomain(domain string) []*index.Document {
	return append([]*index.Document{}, s.byDomain[domain]...)
}

// Definitions returns every definition with key.
func (s Snapshot) Definitions(key string) []DefinitionLocation {
	return append([]DefinitionLocation{}, s.definitions[key]...)
}

// Occurrences returns every occurrence with key.
func (s Snapshot) Occurrences(key string) []OccurrenceLocation {
	return append([]OccurrenceLocation{}, s.occurrences[key]...)
}

// HasDefinition reports whether key resolves to a workspace definition.
func (s Snapshot) HasDefinition(key string) bool { return len(s.definitions[key]) > 0 }
