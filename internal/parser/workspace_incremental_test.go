package parser

import (
	"context"
	"errors"
	"testing"
)

func TestWorkspaceAnalyzerReusesUnaffectedDomains(t *testing.T) {
	analyzer := NewWorkspaceAnalyzer()
	sources := []Source{
		{Path: "/workspace/user.skel", Content: []byte("domain demo.user\npub data User { id: string }\n")},
		{Path: "/workspace/order.skel", Content: []byte("domain demo.order\nimport demo.user\ndata Order { user: user.User }\n")},
		{Path: "/workspace/audit.skel", Content: []byte("domain demo.audit\ndata Audit { id: string }\n")},
	}
	if diagnostics := analyzer.Analyze(sources); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if stats := analyzer.Stats(); stats.ParsedSources != 3 || stats.AnalyzedDomains != 3 {
		t.Fatalf("unexpected initial stats: %+v", stats)
	}

	if diagnostics := analyzer.Analyze(sources); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	if stats := analyzer.Stats(); stats.ReusedSources != 3 || stats.ReusedDomains != 3 {
		t.Fatalf("unexpected reuse stats: %+v", stats)
	}

	sources[0].Content = []byte("domain demo.user\npub data User { id: uuid }\n")
	if diagnostics := analyzer.Analyze(sources); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diagnostics)
	}
	stats := analyzer.Stats()
	if stats.ParsedSources != 1 || stats.ReusedSources != 2 || stats.AnalyzedDomains != 2 || stats.ReusedDomains != 1 {
		t.Fatalf("expected changed domain and dependent only, got %+v", stats)
	}
}

func TestWorkspaceAnalyzerHonorsCancellation(t *testing.T) {
	analyzer := NewWorkspaceAnalyzer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := analyzer.AnalyzeContext(ctx, []Source{{
		Path:    "/workspace/user.skel",
		Content: []byte("domain demo.user\n"),
	}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
