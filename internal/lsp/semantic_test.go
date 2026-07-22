package lsp

import (
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
	diagnostics := semanticDiagnostics(sources, paths)

	require.Len(t, diagnostics, 1)
	require.Len(t, diagnostics[orderURI], 1)
	diagnostic := diagnostics[orderURI][0]
	assert.Equal(t, protocol.String(parser.DiagnosticCodeSemantic), diagnostic.Code)
	assert.Equal(t, protocol.Position{Line: 2, Character: 20}, diagnostic.Range.Start)
	assert.Contains(t, diagnostic.Message, "definition of user.Missing not found")
}

func TestSemanticDiagnosticsDoNotDuplicateSyntaxErrors(t *testing.T) {
	documentURI := uri.File("/workspace/user.skel")
	document := indexDocument(documentURI, "/workspace/user.skel", "domain demo.user\ndata User {", 2)
	sources, paths := semanticSources(map[uri.URI]*_Document{documentURI: document})

	assert.Empty(t, semanticDiagnostics(sources, paths))
}
