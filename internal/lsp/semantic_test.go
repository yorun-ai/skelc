package lsp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/parser"
)

func TestSemanticDiagnosticsAnalyzeInMemoryWorkspace(t *testing.T) {
	userURI := uri.File("/workspace/user.skel")
	orderURI := uri.File("/workspace/order.skel")
	documents := map[uri.URI]*_Document{
		userURI:  indexDocument(userURI, "/workspace/user.skel", "domain demo.user\ndata User {}\n", 2),
		orderURI: indexDocument(orderURI, "/workspace/order.skel", "domain demo.order\nimport demo.user\ndata Order { owner: user.Missing }\n", 7),
	}

	sources, paths := semanticSources(documents)
	diagnostics, err := semanticDiagnostics(context.Background(), parser.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)

	require.Len(t, diagnostics, 1)
	require.Len(t, diagnostics[orderURI], 1)
	diagnostic := diagnostics[orderURI][0]
	assert.Equal(t, protocol.String(parser.DiagnosticCodeSemanticReference), diagnostic.Code)
	assert.Equal(t, protocol.Position{Line: 2, Character: 20}, diagnostic.Range.Start)
	assert.Equal(t, protocol.Position{Line: 2, Character: 32}, diagnostic.Range.End)
	assert.Contains(t, diagnostic.Message, "definition of user.Missing not found")
}

func TestSemanticDiagnosticsDoNotDuplicateSyntaxErrors(t *testing.T) {
	documentURI := uri.File("/workspace/user.skel")
	document := indexDocument(documentURI, "/workspace/user.skel", "domain demo.user\ndata User {", 2)
	sources, paths := semanticSources(map[uri.URI]*_Document{documentURI: document})

	diagnostics, err := semanticDiagnostics(context.Background(), parser.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}

func TestSemanticDiagnosticsPublishMultipleErrorsForOneDocument(t *testing.T) {
	documentURI := uri.File("/workspace/data.skel")
	document := indexDocument(documentURI, "/workspace/data.skel", `domain demo
data User { missing: MissingUser }
data Order { missing: MissingOrder }
`, 3)
	sources, paths := semanticSources(map[uri.URI]*_Document{documentURI: document})

	diagnostics, err := semanticDiagnostics(context.Background(), parser.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)

	require.Len(t, diagnostics[documentURI], 2)
	assert.Contains(t, diagnostics[documentURI][0].Message, "MissingUser")
	assert.Contains(t, diagnostics[documentURI][1].Message, "MissingOrder")
}

func TestSemanticDiagnosticsIncludeDuplicateRelatedLocation(t *testing.T) {
	documentURI := uri.File("/workspace/data.skel")
	document := indexDocument(documentURI, "/workspace/data.skel", "domain demo\ndata User {}\ndata User {}\n", 1)
	sources, paths := semanticSources(map[uri.URI]*_Document{documentURI: document})

	diagnostics, err := semanticDiagnostics(context.Background(), parser.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	require.Len(t, diagnostics[documentURI], 1)
	diagnostic := diagnostics[documentURI][0]
	assert.Equal(t, protocol.String(parser.DiagnosticCodeSemanticDuplicate), diagnostic.Code)
	require.Len(t, diagnostic.RelatedInformation, 1)
	assert.Equal(t, protocol.Position{Line: 1, Character: 5}, diagnostic.RelatedInformation[0].Location.Range.Start)
}
