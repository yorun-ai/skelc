// Package lsp implements the Skel language server used by editor integrations.
package lsp

import (
	"context"
	"io"
	"sync"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type _Server struct {
	protocol.UnimplementedServer

	mu            sync.RWMutex
	documents     map[uri.URI]*_Document
	open          map[uri.URI]bool
	semantic      map[uri.URI][]protocol.Diagnostic
	client        protocol.Client
	generation    uint64
	semanticTimer *time.Timer
	exit          chan struct{}
	exitOnce      sync.Once
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
	return &_Server{
		documents: map[uri.URI]*_Document{}, open: map[uri.URI]bool{},
		semantic: map[uri.URI][]protocol.Diagnostic{}, exit: make(chan struct{}),
	}
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
	prepareRename := true
	change := protocol.TextDocumentSyncKindFull
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			PositionEncoding:           protocol.PositionEncodingKindUTF16,
			TextDocumentSync:           &protocol.TextDocumentSyncOptions{OpenClose: &openClose, Change: &change},
			CompletionProvider:         &protocol.CompletionOptions{TriggerCharacters: []string{"."}},
			HoverProvider:              protocol.Boolean(true),
			DefinitionProvider:         protocol.Boolean(true),
			ReferencesProvider:         protocol.Boolean(true),
			DocumentSymbolProvider:     protocol.Boolean(true),
			WorkspaceSymbolProvider:    protocol.Boolean(true),
			DocumentFormattingProvider: protocol.Boolean(true),
			RenameProvider:             &protocol.RenameOptions{PrepareProvider: &prepareRename},
		},
		ServerInfo: protocol.ServerInfo{Name: "skelc"},
	}, nil
}

func (s *_Server) Initialized(ctx context.Context, _ *protocol.InitializedParams) error {
	s.rememberClient(ctx)
	s.scheduleSemanticAnalysis()
	return nil
}

func (s *_Server) Shutdown(context.Context) error {
	s.stopSemanticAnalysis()
	return nil
}

func (s *_Server) Exit(context.Context) error {
	s.stopSemanticAnalysis()
	s.exitOnce.Do(func() { close(s.exit) })
	return nil
}
