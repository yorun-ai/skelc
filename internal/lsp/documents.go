package lsp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
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
	if content, err := os.ReadFile(documentURI.FsPath()); err == nil && s.workspaceDocumentTrackedLocked(documentURI) {
		s.documents[documentURI] = indexDocument(documentURI, documentURI.FsPath(), string(content), 0)
		exists = true
	} else {
		delete(s.documents, documentURI)
		s.untrackWorkspaceDocumentLocked(documentURI)
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
			s.untrackWorkspaceDocumentLocked(documentURI)
			s.mu.Unlock()
			continue
		}
		content, err := os.ReadFile(documentURI.FsPath())
		if err == nil {
			s.documents[documentURI] = indexDocument(documentURI, documentURI.FsPath(), string(content), 0)
			s.trackWorkspaceDocumentLocked(documentURI)
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

func (s *_Server) DidChangeWorkspaceFolders(ctx context.Context, params *protocol.DidChangeWorkspaceFoldersParams) error {
	removed := make([]uri.URI, 0)
	for _, folder := range params.Event.Removed {
		removed = append(removed, s.removeWorkspace(folder.URI)...)
	}
	for _, folder := range params.Event.Added {
		s.loadWorkspace(folder.URI)
	}
	s.invalidateSemanticDiagnostics(ctx)
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return nil
	}
	slices.Sort(removed)
	for _, documentURI := range removed {
		s.mu.RLock()
		documentExists := s.documents[documentURI] != nil
		s.mu.RUnlock()
		if documentExists {
			continue
		}
		if err := client.PublishDiagnostics(ctx, &protocol.PublishDiagnosticsParams{
			URI: documentURI, Diagnostics: []protocol.Diagnostic{},
		}); err != nil {
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

func (s *_Server) loadWorkspace(rootURI uri.URI) {
	rootPath := rootURI.FsPath()
	documents := map[uri.URI]*_Document{}
	_ = filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() && path != rootPath && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" || entry.Name() == "vendor") {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".skel" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		documentURI := workspaceDocumentURI(rootURI, rootPath, path)
		documents[documentURI] = indexDocument(documentURI, path, string(content), 0)
		return nil
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	tracked := make(map[uri.URI]struct{}, len(documents))
	for documentURI, document := range documents {
		tracked[documentURI] = struct{}{}
		if !s.open[documentURI] {
			s.documents[documentURI] = document
		}
	}
	s.workspaceFiles[rootURI] = tracked
}

func (s *_Server) removeWorkspace(rootURI uri.URI) []uri.URI {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked := s.workspaceFiles[rootURI]
	delete(s.workspaceFiles, rootURI)
	removed := make([]uri.URI, 0, len(tracked))
	for documentURI := range tracked {
		if s.open[documentURI] || s.workspaceDocumentTrackedLocked(documentURI) {
			continue
		}
		delete(s.documents, documentURI)
		removed = append(removed, documentURI)
	}
	return removed
}

func (s *_Server) trackWorkspaceDocumentLocked(documentURI uri.URI) {
	for rootURI, tracked := range s.workspaceFiles {
		if workspaceContains(rootURI, documentURI) {
			tracked[documentURI] = struct{}{}
		}
	}
}

func (s *_Server) untrackWorkspaceDocumentLocked(documentURI uri.URI) {
	for _, tracked := range s.workspaceFiles {
		delete(tracked, documentURI)
	}
}

func (s *_Server) workspaceDocumentTrackedLocked(documentURI uri.URI) bool {
	for _, tracked := range s.workspaceFiles {
		if _, ok := tracked[documentURI]; ok {
			return true
		}
	}
	return false
}

func workspaceDocumentURI(rootURI uri.URI, rootPath, path string) uri.URI {
	if rootURI.IsFile() {
		return uri.File(path)
	}
	relative, err := filepath.Rel(rootPath, path)
	if err != nil {
		return uri.File(path)
	}
	documentURI, err := uri.JoinPath(rootURI, filepath.ToSlash(relative))
	if err != nil {
		return uri.File(path)
	}
	return documentURI
}

func workspaceContains(rootURI, documentURI uri.URI) bool {
	if rootURI.Scheme() != documentURI.Scheme() || rootURI.Authority() != documentURI.Authority() {
		return false
	}
	relative, err := filepath.Rel(rootURI.FsPath(), documentURI.FsPath())
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
