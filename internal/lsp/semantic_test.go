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

func TestSemanticDiagnosticsDoNotResolveImportsAcrossDomainRoots(t *testing.T) {
	userURI := uri.File("/workspace/user/user.skel")
	orderURI := uri.File("/workspace/order/order.skel")
	documents := map[uri.URI]*_Document{
		userURI:  indexDocument(userURI, userURI.FsPath(), "domain demo.user\ndata User {}\n", 2),
		orderURI: indexDocument(orderURI, orderURI.FsPath(), "domain demo.order\nimport demo.user\ndata Order { owner: user.Missing }\n", 7),
	}

	sources, paths := semanticSources(documents)
	diagnostics, err := semanticDiagnostics(context.Background(), parser.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
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

func TestSemanticDiagnosticsKeepSameNamedDomainDirectoriesIndependent(t *testing.T) {
	sourceURI := uri.File("/workspace/domain/base/skel/actor.skel")
	generatedURI := uri.File("/workspace/domain/base/pub/skeled/skel/types.skel")
	documents := map[uri.URI]*_Document{
		sourceURI: indexDocument(
			sourceURI,
			sourceURI.FsPath(),
			"domain base\npub resource User { action read }\n",
			1,
		),
		generatedURI: indexDocument(
			generatedURI,
			generatedURI.FsPath(),
			"domain base\npub resource User { action read }\n",
			1,
		),
	}

	sources, paths := semanticSources(documents)
	diagnostics, err := semanticDiagnostics(context.Background(), parser.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}

func TestSemanticDiagnosticsMergeSameNamedDomainFilesInOneDirectory(t *testing.T) {
	firstURI := uri.File("/workspace/domain/base/skel/first.skel")
	secondURI := uri.File("/workspace/domain/base/skel/second.skel")
	documents := map[uri.URI]*_Document{
		firstURI: indexDocument(
			firstURI,
			firstURI.FsPath(),
			"domain base\ndata User {}\n",
			1,
		),
		secondURI: indexDocument(
			secondURI,
			secondURI.FsPath(),
			"domain base\ndata User {}\n",
			1,
		),
	}

	sources, paths := semanticSources(documents)
	diagnostics, err := semanticDiagnostics(context.Background(), parser.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	require.Len(t, diagnostics[secondURI], 1)
	assert.Equal(t, protocol.String(parser.DiagnosticCodeSemanticDuplicate), diagnostics[secondURI][0].Code)
}
