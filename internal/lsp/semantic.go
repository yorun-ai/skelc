package lsp

import (
	"context"
	"path/filepath"
	"slices"
	"time"

	"github.com/alecthomas/participle/v2/lexer"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/parser"
)

const semanticAnalysisDelay = 200 * time.Millisecond

func (s *_Server) stopSemanticAnalysis() {
	s.mu.Lock()
	s.generation++
	if s.semanticTimer != nil {
		s.semanticTimer.Stop()
		s.semanticTimer = nil
	}
	s.mu.Unlock()
}

func (s *_Server) rememberClient(ctx context.Context) {
	client, ok := protocol.ClientFromContext(ctx)
	if !ok {
		return
	}
	s.mu.Lock()
	s.client = client
	s.mu.Unlock()
}

// invalidateSemanticDiagnostics removes results produced from an older
// workspace snapshot before scheduling a replacement. Syntax diagnostics are
// retained because they are computed directly from the current document.
func (s *_Server) invalidateSemanticDiagnostics(ctx context.Context) {
	s.rememberClient(ctx)
	s.mu.Lock()
	stale := make([]uri.URI, 0, len(s.semantic))
	for documentURI := range s.semantic {
		stale = append(stale, documentURI)
	}
	s.semantic = map[uri.URI][]protocol.Diagnostic{}
	s.scheduleSemanticAnalysisLocked()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return
	}
	slices.Sort(stale)
	for _, documentURI := range stale {
		_ = s.publishDiagnosticsWithClient(ctx, client, documentURI)
	}
}

func (s *_Server) scheduleSemanticAnalysis() {
	s.mu.Lock()
	s.scheduleSemanticAnalysisLocked()
	s.mu.Unlock()
}

func (s *_Server) scheduleSemanticAnalysisLocked() {
	s.generation++
	generation := s.generation
	if s.semanticTimer != nil {
		s.semanticTimer.Stop()
	}
	if s.client == nil {
		s.semanticTimer = nil
		return
	}
	s.semanticTimer = time.AfterFunc(semanticAnalysisDelay, func() {
		s.runSemanticAnalysis(generation)
	})
}

func (s *_Server) runSemanticAnalysis(generation uint64) {
	s.mu.RLock()
	if generation != s.generation {
		s.mu.RUnlock()
		return
	}
	sources, paths := semanticSources(s.documents)
	client := s.client
	s.mu.RUnlock()

	semantic := semanticDiagnostics(sources, paths)

	s.mu.Lock()
	if generation != s.generation || client == nil {
		s.mu.Unlock()
		return
	}
	previous := s.semantic
	s.semantic = semantic
	s.semanticTimer = nil
	changed := make(map[uri.URI]bool, len(previous)+len(semantic))
	for documentURI := range previous {
		changed[documentURI] = true
	}
	for documentURI := range semantic {
		changed[documentURI] = true
	}
	s.mu.Unlock()

	documentURIs := make([]uri.URI, 0, len(changed))
	for documentURI := range changed {
		documentURIs = append(documentURIs, documentURI)
	}
	slices.Sort(documentURIs)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, documentURI := range documentURIs {
		_ = s.publishDiagnosticsWithClient(ctx, client, documentURI)
	}
}

func semanticSources(documents map[uri.URI]*_Document) ([]parser.Source, map[string]uri.URI) {
	sources := make([]parser.Source, 0, len(documents))
	paths := make(map[string]uri.URI, len(documents))
	for documentURI, document := range documents {
		path := filepath.Clean(document.Path)
		sources = append(sources, parser.Source{
			Path: path, Domain: document.Domain, Content: []byte(document.Source),
		})
		paths[path] = documentURI
	}
	return sources, paths
}

func semanticDiagnostics(sources []parser.Source, paths map[string]uri.URI) map[uri.URI][]protocol.Diagnostic {
	result := map[uri.URI][]protocol.Diagnostic{}
	contents := make(map[string]string, len(sources))
	for _, source := range sources {
		contents[filepath.Clean(source.Path)] = string(source.Content)
	}
	for _, diagnostic := range parser.AnalyzeWorkspace(sources) {
		if diagnostic.Code == parser.DiagnosticCodeSyntax {
			continue
		}
		documentURI, ok := paths[filepath.Clean(diagnostic.Position.File)]
		if !ok {
			continue
		}
		source := contents[filepath.Clean(diagnostic.Position.File)]
		range_ := identifierRange(source, lexer.Position{
			Filename: diagnostic.Position.File, Line: diagnostic.Position.Line, Column: diagnostic.Position.Column,
		}, "")
		if range_.End == range_.Start {
			range_.End.Character++
		}
		result[documentURI] = append(result[documentURI], protocol.Diagnostic{
			Range: range_, Severity: protocol.DiagnosticSeverityError,
			Code: protocol.String(diagnostic.Code), Source: protocol.NewOptional("skelc"),
			Message: protocol.String(diagnostic.Message),
		})
	}
	return result
}
