// Package workspace owns the mutable set of documents visible to the language
// server and exposes immutable snapshots for analysis and language features.
package workspace

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/lsp/index"
)

// Store owns open documents, workspace roots, and on-disk document indexes.
type Store struct {
	mu             sync.RWMutex
	documents      map[uri.URI]*index.Document
	open           map[uri.URI]bool
	workspaceFiles map[uri.URI]map[uri.URI]struct{}
	revision       uint64
}

// New creates an empty workspace store.
func New() *Store {
	return &Store{
		documents:      map[uri.URI]*index.Document{},
		open:           map[uri.URI]bool{},
		workspaceFiles: map[uri.URI]map[uri.URI]struct{}{},
	}
}

// Put indexes an in-memory document. Open documents take precedence over
// workspace files loaded from disk.
func (s *Store) Put(documentURI uri.URI, content string, version int32, open bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.documents[documentURI] = index.Build(documentURI, documentURI.FsPath(), content, version)
	if open {
		s.open[documentURI] = true
	}
	s.revision++
}

// Close closes an editor document and restores its on-disk contents when it
// remains tracked by a workspace root. It reports whether the document still
// exists in the store.
func (s *Store) Close(documentURI uri.URI) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.open, documentURI)
	if content, err := os.ReadFile(documentURI.FsPath()); err == nil && s.documentTrackedLocked(documentURI) {
		s.documents[documentURI] = index.Build(documentURI, documentURI.FsPath(), string(content), 0)
		s.revision++
		return true
	}
	delete(s.documents, documentURI)
	s.untrackLocked(documentURI)
	s.revision++
	return false
}

// ApplyFileChanges reloads non-open workspace files and returns the affected
// document URIs in notification order.
func (s *Store) ApplyFileChanges(changes []protocol.FileEvent) []uri.URI {
	changed := make([]uri.URI, 0, len(changes))
	for _, change := range changes {
		documentURI := change.URI
		s.mu.Lock()
		if s.open[documentURI] {
			s.mu.Unlock()
			continue
		}
		changed = append(changed, documentURI)
		if change.Type == protocol.FileChangeTypeDeleted {
			delete(s.documents, documentURI)
			s.untrackLocked(documentURI)
			s.revision++
			s.mu.Unlock()
			continue
		}
		if content, err := os.ReadFile(documentURI.FsPath()); err == nil {
			s.documents[documentURI] = index.Build(documentURI, documentURI.FsPath(), string(content), 0)
			s.trackLocked(documentURI)
			s.revision++
		}
		s.mu.Unlock()
	}
	return changed
}

// AddRoot discovers and indexes Skel files below a workspace root.
func (s *Store) AddRoot(rootURI uri.URI) {
	rootPath := rootURI.FsPath()
	documents := map[uri.URI]*index.Document{}
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
		documentURI := documentURI(rootURI, rootPath, path)
		documents[documentURI] = index.Build(documentURI, path, string(content), 0)
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
	s.revision++
}

// RemoveRoot removes documents that are no longer open or tracked by another
// workspace root and returns the removed URIs.
func (s *Store) RemoveRoot(rootURI uri.URI) []uri.URI {
	s.mu.Lock()
	defer s.mu.Unlock()
	tracked := s.workspaceFiles[rootURI]
	delete(s.workspaceFiles, rootURI)
	removed := make([]uri.URI, 0, len(tracked))
	for documentURI := range tracked {
		if s.open[documentURI] || s.documentTrackedLocked(documentURI) {
			continue
		}
		delete(s.documents, documentURI)
		removed = append(removed, documentURI)
	}
	s.revision++
	return removed
}

// Snapshot returns an immutable view of the current workspace.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	documents := make(map[uri.URI]*index.Document, len(s.documents))
	ordered := make([]*index.Document, 0, len(s.documents))
	for documentURI, document := range s.documents {
		documents[documentURI] = document
		ordered = append(ordered, document)
	}
	slices.SortFunc(ordered, func(left, right *index.Document) int {
		return strings.Compare(string(left.URI), string(right.URI))
	})
	return newSnapshot(s.revision, documents, ordered)
}

func (s *Store) trackLocked(documentURI uri.URI) {
	for rootURI, tracked := range s.workspaceFiles {
		if contains(rootURI, documentURI) {
			tracked[documentURI] = struct{}{}
		}
	}
}

func (s *Store) untrackLocked(documentURI uri.URI) {
	for _, tracked := range s.workspaceFiles {
		delete(tracked, documentURI)
	}
}

func (s *Store) documentTrackedLocked(documentURI uri.URI) bool {
	for _, tracked := range s.workspaceFiles {
		if _, ok := tracked[documentURI]; ok {
			return true
		}
	}
	return false
}

func documentURI(rootURI uri.URI, rootPath, path string) uri.URI {
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

func contains(rootURI, documentURI uri.URI) bool {
	if rootURI.Scheme() != documentURI.Scheme() || rootURI.Authority() != documentURI.Authority() {
		return false
	}
	relative, err := filepath.Rel(rootURI.FsPath(), documentURI.FsPath())
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
