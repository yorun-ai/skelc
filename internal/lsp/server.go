// Package lsp implements the Skel language server used by editor integrations.
package lsp

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/participle/v2"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type _Server struct {
	protocol.UnimplementedServer

	mu        sync.RWMutex
	documents map[uri.URI]*_Document
	open      map[uri.URI]bool
	exit      chan struct{}
	exitOnce  sync.Once
}

type _ReadWriteCloser struct {
	io.Reader
	io.Writer
	closer io.Closer
}

func (rw *_ReadWriteCloser) Close() error {
	if rw.closer != nil {
		return rw.closer.Close()
	}
	return nil
}

// Serve runs a Language Server Protocol connection over the supplied streams.
func Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	server := newServer()
	closer, _ := input.(io.Closer)
	stream := jsonrpc2.NewStream(&_ReadWriteCloser{Reader: input, Writer: output, closer: closer})
	_, connection, _ := protocol.NewServer(ctx, server, stream)
	select {
	case <-connection.Done():
	case <-server.exit:
		if err := connection.Close(); err != nil {
			return err
		}
	}
	return connection.Err()
}

func newServer() *_Server {
	return &_Server{documents: map[uri.URI]*_Document{}, open: map[uri.URI]bool{}, exit: make(chan struct{})}
}

func (s *_Server) Initialize(_ context.Context, params *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	if folders, ok := params.WorkspaceFolders.Get(); ok {
		for _, folder := range folders {
			s.loadWorkspace(folder.URI.FsPath())
		}
	} else if params.RootURI != nil {
		s.loadWorkspace(params.RootURI.FsPath())
	} else if rootPath, ok := params.RootPath.Get(); ok {
		s.loadWorkspace(rootPath)
	}
	openClose := true
	change := protocol.TextDocumentSyncKindFull
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			PositionEncoding:       protocol.PositionEncodingKindUTF16,
			TextDocumentSync:       &protocol.TextDocumentSyncOptions{OpenClose: &openClose, Change: &change},
			DefinitionProvider:     protocol.Boolean(true),
			ReferencesProvider:     protocol.Boolean(true),
			DocumentSymbolProvider: protocol.Boolean(true),
		},
		ServerInfo: protocol.ServerInfo{Name: "skelc"},
	}, nil
}

func (s *_Server) Initialized(context.Context, *protocol.InitializedParams) error { return nil }

func (s *_Server) Shutdown(context.Context) error { return nil }

func (s *_Server) Exit(context.Context) error {
	s.exitOnce.Do(func() { close(s.exit) })
	return nil
}

func (s *_Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	document := params.TextDocument
	s.putDocument(document.URI, document.Text, document.Version, true)
	return s.publishDiagnostics(ctx, document.URI)
}

func (s *_Server) DidChange(ctx context.Context, params *protocol.DidChangeTextDocumentParams) error {
	if len(params.ContentChanges) == 0 {
		return nil
	}
	change, ok := params.ContentChanges[len(params.ContentChanges)-1].(*protocol.TextDocumentContentChangeWholeDocument)
	if !ok {
		return errors.New("skelc lsp requires full document synchronization")
	}
	s.putDocument(params.TextDocument.URI, change.Text, params.TextDocument.Version, true)
	return s.publishDiagnostics(ctx, params.TextDocument.URI)
}

func (s *_Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	documentURI := params.TextDocument.URI
	s.mu.Lock()
	delete(s.open, documentURI)
	if content, err := os.ReadFile(documentURI.FsPath()); err == nil {
		s.documents[documentURI] = indexDocument(documentURI, documentURI.FsPath(), string(content), 0)
	} else {
		delete(s.documents, documentURI)
	}
	s.mu.Unlock()
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return nil
	}
	return client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{URI: documentURI, Diagnostics: []protocol.Diagnostic{}})
}

func (s *_Server) DidChangeWatchedFiles(_ context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	for _, change := range params.Changes {
		documentURI := change.URI
		s.mu.Lock()
		if s.open[documentURI] {
			s.mu.Unlock()
			continue
		}
		if change.Type == protocol.FileChangeTypeDeleted {
			delete(s.documents, documentURI)
			s.mu.Unlock()
			continue
		}
		content, err := os.ReadFile(documentURI.FsPath())
		if err == nil {
			s.documents[documentURI] = indexDocument(documentURI, documentURI.FsPath(), string(content), 0)
		}
		s.mu.Unlock()
	}
	return nil
}

func (s *_Server) Definition(_ context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return protocol.LocationSlice{}, nil
	}
	occurrence, ok := occurrenceAt(document, params.Position)
	if !ok {
		return protocol.LocationSlice{}, nil
	}
	locations := make([]protocol.Location, 0)
	for _, candidate := range s.documents {
		for _, definition := range candidate.Definitions {
			if definition.Key == occurrence.Key {
				locations = append(locations, protocol.Location{URI: candidate.URI, Range: definition.Range})
			}
		}
	}
	sortLocations(locations)
	return protocol.LocationSlice(locations), nil
}

func (s *_Server) References(_ context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return []protocol.Location{}, nil
	}
	occurrence, ok := occurrenceAt(document, params.Position)
	if !ok {
		return []protocol.Location{}, nil
	}
	locations := make([]protocol.Location, 0)
	definitions := make(map[protocol.Location]bool)
	for _, candidate := range s.documents {
		for _, definition := range candidate.Definitions {
			if definition.Key == occurrence.Key {
				definitions[protocol.Location{URI: candidate.URI, Range: definition.Range}] = true
			}
		}
	}
	for _, candidate := range s.documents {
		for _, reference := range candidate.Occurrences {
			if reference.Key == occurrence.Key {
				location := protocol.Location{URI: candidate.URI, Range: reference.Range}
				if params.Context.IncludeDeclaration || !definitions[location] {
					locations = append(locations, location)
				}
			}
		}
	}
	sortLocations(locations)
	return locations, nil
}

func (s *_Server) DocumentSymbol(_ context.Context, params *protocol.DocumentSymbolParams) (protocol.DocumentSymbolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	document := s.documents[params.TextDocument.URI]
	if document == nil {
		return protocol.DocumentSymbolSlice{}, nil
	}
	symbols := make(protocol.DocumentSymbolSlice, 0, len(document.Definitions))
	for _, definition := range document.Definitions {
		symbols = append(symbols, protocol.DocumentSymbol{
			Name: definition.Name, Kind: definition.Kind, Range: definition.Range, SelectionRange: definition.Range,
		})
	}
	return symbols, nil
}

func (s *_Server) putDocument(documentURI uri.URI, source string, version int32, open bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[documentURI] = indexDocument(documentURI, documentURI.FsPath(), source, version)
	if open {
		s.open[documentURI] = true
	}
}

func (s *_Server) loadWorkspace(root string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() && path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".skel" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		documentURI := uri.File(path)
		s.documents[documentURI] = indexDocument(documentURI, path, string(content), 0)
		return nil
	})
}

func (s *_Server) publishDiagnostics(ctx context.Context, documentURI uri.URI) error {
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return nil
	}
	s.mu.RLock()
	document := s.documents[documentURI]
	s.mu.RUnlock()
	diagnostics := []protocol.Diagnostic{}
	if document != nil && document.ParseError != nil {
		position := protocol.Position{}
		message := document.ParseError.Error()
		var parseError participle.Error
		if errors.As(document.ParseError, &parseError) {
			parsePosition := parseError.Position()
			position = identifierRange(document.Source, parsePosition, "").Start
			message = parseError.Message()
		}
		diagnostics = append(diagnostics, protocol.Diagnostic{
			Range: protocol.Range{Start: position, End: protocol.Position{Line: position.Line, Character: position.Character + 1}}, Severity: protocol.DiagnosticSeverityError,
			Source: protocol.NewOptional("skelc"), Message: protocol.String(message),
		})
	}
	params := &protocol.PublishDiagnosticsParams{URI: documentURI, Diagnostics: diagnostics}
	if document != nil {
		params.Version = protocol.NewOptional(document.Version)
	}
	return client.PublishDiagnostics(ctx, params)
}
