package analysis

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/compiler"
	"go.yorun.ai/skelc/internal/lsp/index"
)

func TestSemanticDiagnosticsDoNotResolveImportsAcrossDomainRoots(t *testing.T) {
	userURI := uri.File("/workspace/user/user.skel")
	orderURI := uri.File("/workspace/order/order.skel")
	documents := map[uri.URI]*index.Document{
		userURI:  index.Build(userURI, userURI.FsPath(), "domain demo.user\ndata User {}\n", 2),
		orderURI: index.Build(orderURI, orderURI.FsPath(), "domain demo.order\nimport demo.user\ndata Order { owner: user.Missing }\n", 7),
	}

	sources, paths := SemanticSources(documents)
	diagnostics, err := SemanticDiagnostics(context.Background(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}

func TestSemanticDiagnosticsDoNotDuplicateSyntaxErrors(t *testing.T) {
	documentURI := uri.File("/workspace/user.skel")
	document := index.Build(documentURI, "/workspace/user.skel", "domain demo.user\ndata User {", 2)
	sources, paths := SemanticSources(map[uri.URI]*index.Document{documentURI: document})

	diagnostics, err := SemanticDiagnostics(context.Background(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}

func TestSemanticDiagnosticsPublishMultipleErrorsForOneDocument(t *testing.T) {
	documentURI := uri.File("/workspace/data.skel")
	document := index.Build(documentURI, "/workspace/data.skel", `domain demo
data User { missing: MissingUser }
data Order { missing: MissingOrder }
`, 3)
	sources, paths := SemanticSources(map[uri.URI]*index.Document{documentURI: document})

	diagnostics, err := SemanticDiagnostics(context.Background(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)

	require.Len(t, diagnostics[documentURI], 2)
	assert.Contains(t, diagnostics[documentURI][0].Message, "MissingUser")
	assert.Contains(t, diagnostics[documentURI][1].Message, "MissingOrder")
}

func TestSemanticDiagnosticsIncludeDuplicateRelatedLocation(t *testing.T) {
	documentURI := uri.File("/workspace/data.skel")
	document := index.Build(documentURI, "/workspace/data.skel", "domain demo\ndata User {}\ndata User {}\n", 1)
	sources, paths := SemanticSources(map[uri.URI]*index.Document{documentURI: document})

	diagnostics, err := SemanticDiagnostics(context.Background(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	require.Len(t, diagnostics[documentURI], 1)
	diagnostic := diagnostics[documentURI][0]
	assert.Equal(t, protocol.String(compiler.DiagnosticCodeSemanticDuplicate), diagnostic.Code)
	require.Len(t, diagnostic.RelatedInformation, 1)
	assert.Equal(t, protocol.Position{Line: 1, Character: 5}, diagnostic.RelatedInformation[0].Location.Range.Start)
}

func TestSemanticDiagnosticsKeepSameNamedDomainDirectoriesIndependent(t *testing.T) {
	sourceURI := uri.File("/workspace/domain/base/skel/actor.skel")
	generatedURI := uri.File("/workspace/domain/base/pub/skeled/skel/types.skel")
	documents := map[uri.URI]*index.Document{
		sourceURI: index.Build(
			sourceURI,
			sourceURI.FsPath(),
			"domain base\npub resource User { action read }\n",
			1,
		),
		generatedURI: index.Build(
			generatedURI,
			generatedURI.FsPath(),
			"domain base\npub resource User { action read }\n",
			1,
		),
	}

	sources, paths := SemanticSources(documents)
	diagnostics, err := SemanticDiagnostics(context.Background(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
}

func TestSemanticDiagnosticsMergeSameNamedDomainFilesWithDomainFile(t *testing.T) {
	firstURI := uri.File("/workspace/domain/base/skel/first.skel")
	secondURI := uri.File("/workspace/domain/base/skel/second.skel")
	documents := map[uri.URI]*index.Document{
		firstURI: index.Build(
			firstURI,
			firstURI.FsPath(),
			"domain base\ndata User {}\n",
			1,
		),
		secondURI: index.Build(
			secondURI,
			secondURI.FsPath(),
			"domain base\ndata User {}\n",
			1,
		),
	}

	domainURI := uri.File("/workspace/domain/base/skel/domain.skel")
	documents[domainURI] = index.Build(domainURI, domainURI.FsPath(), "domain base\n", 1)

	sources, paths := SemanticSources(documents)
	diagnostics, err := SemanticDiagnostics(context.Background(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	require.Len(t, diagnostics[secondURI], 1)
	assert.Equal(t, protocol.String(compiler.DiagnosticCodeSemanticDuplicate), diagnostics[secondURI][0].Code)
}

func TestSemanticDiagnosticsKeepStandaloneFormatterFixturesIndependent(t *testing.T) {
	documents := map[uri.URI]*index.Document{}
	for _, name := range []string{"complete.input.skel", "complete.golden.skel"} {
		path, err := filepath.Abs(filepath.Join("../../formatter/testdata", name))
		require.NoError(t, err)
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		documentURI := uri.File(path)
		documents[documentURI] = index.Build(documentURI, path, string(content), 1)
	}
	sources, paths := SemanticSources(documents)
	diagnostics, domains, err := SemanticWorkspace(t.Context(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	assert.Empty(t, diagnostics)
	require.Len(t, domains, 2)
	for _, domain := range domains {
		require.Len(t, domain.Sources, 1)
		assert.Equal(t, domain.Sources[0].Path, domain.Root)
	}
}

func TestSemanticDiagnosticsChangeGroupingWithDomainFile(t *testing.T) {
	firstURI := uri.File("/workspace/first.skel")
	secondURI := uri.File("/workspace/second.skel")
	domainURI := uri.File("/workspace/domain.skel")
	documents := map[uri.URI]*index.Document{
		firstURI:  index.Build(firstURI, firstURI.FsPath(), "domain demo\ndata User {}\n", 1),
		secondURI: index.Build(secondURI, secondURI.FsPath(), "domain demo\ndata Order { user: User }\n", 1),
	}
	analyzer := compiler.NewWorkspaceAnalyzer()
	for _, hasDomainFile := range []bool{false, true, false} {
		if hasDomainFile {
			documents[domainURI] = index.Build(domainURI, domainURI.FsPath(), "domain demo\n", 1)
		} else {
			delete(documents, domainURI)
		}
		sources, paths := SemanticSources(documents)
		diagnostics, err := SemanticDiagnostics(t.Context(), analyzer, sources, paths)
		require.NoError(t, err)
		if hasDomainFile {
			assert.Empty(t, diagnostics)
		} else {
			require.Len(t, diagnostics[secondURI], 1)
			assert.Contains(t, diagnostics[secondURI][0].Message, "User")
		}
	}
}
