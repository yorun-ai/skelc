package lsp

import (
	"context"
	"slices"
	"time"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

const semanticAnalysisDelay = 200 * time.Millisecond

func (s *_Server) stopSemanticAnalysis() {
	s.mu.Lock()
	s.generation++
	if s.semanticTimer != nil {
		s.semanticTimer.Stop()
		s.semanticTimer = nil
	}
	if s.semanticCancel != nil {
		s.semanticCancel()
		s.semanticCancel = nil
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
	if s.semanticCancel != nil {
		s.semanticCancel()
	}
	analysisContext, cancel := context.WithCancel(context.Background())
	s.semanticCancel = cancel
	if s.client == nil {
		cancel()
		s.semanticCancel = nil
		s.semanticTimer = nil
		return
	}
	s.semanticTimer = time.AfterFunc(semanticAnalysisDelay, func() {
		s.runSemanticAnalysis(analysisContext, generation)
	})
}

func (s *_Server) runSemanticAnalysis(analysisContext context.Context, generation uint64) {
	s.mu.RLock()
	if generation != s.generation {
		s.mu.RUnlock()
		return
	}
	sources, paths := semanticSources(s.documents)
	client := s.client
	s.mu.RUnlock()

	semantic, err := semanticDiagnostics(analysisContext, s.analyzer, sources, paths)
	if err != nil {
		return
	}

	s.mu.Lock()
	if generation != s.generation || client == nil {
		s.mu.Unlock()
		return
	}
	previous := s.semantic
	s.semantic = semantic
	s.semanticTimer = nil
	s.semanticCancel = nil
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
