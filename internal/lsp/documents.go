package lsp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func (s *_Server) DidOpen(ctx context.Context, params *protocol.DidOpenTextDocumentParams) error {
	document := params.TextDocument
	s.putDocument(document.URI, document.Text, document.Version, true)
	s.invalidateSemanticDiagnostics(ctx)
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
	s.invalidateSemanticDiagnostics(ctx)
	return s.publishDiagnostics(ctx, params.TextDocument.URI)
}

func (s *_Server) DidClose(ctx context.Context, params *protocol.DidCloseTextDocumentParams) error {
	documentURI := params.TextDocument.URI
	s.mu.Lock()
	delete(s.open, documentURI)
	exists := false
	if content, err := os.ReadFile(documentURI.FsPath()); err == nil {
		s.documents[documentURI] = indexDocument(documentURI, documentURI.FsPath(), string(content), 0)
		exists = true
	} else {
		delete(s.documents, documentURI)
	}
	s.mu.Unlock()
	s.invalidateSemanticDiagnostics(ctx)
	if exists {
		return s.publishDiagnostics(ctx, documentURI)
	}
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return nil
	}
	return client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{URI: documentURI, Diagnostics: []protocol.Diagnostic{}})
}

func (s *_Server) DidChangeWatchedFiles(ctx context.Context, params *protocol.DidChangeWatchedFilesParams) error {
	changed := make([]uri.URI, 0, len(params.Changes))
	for _, change := range params.Changes {
		documentURI := change.URI
		s.mu.Lock()
		if s.open[documentURI] {
			s.mu.Unlock()
			continue
		}
		changed = append(changed, documentURI)
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
	s.invalidateSemanticDiagnostics(ctx)
	for _, documentURI := range changed {
		if err := s.publishDiagnostics(ctx, documentURI); err != nil {
			return err
		}
	}
	return nil
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
