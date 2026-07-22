package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestServerReturnsNestedDocumentAndWorkspaceSymbols(t *testing.T) {
	server := newServer()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, `domain demo
data User {
    id: int
}
`, 1, true)

	documentResult, err := server.DocumentSymbol(t.Context(), &protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
	})
	require.NoError(t, err)
	documentSymbols := documentResult.(protocol.DocumentSymbolSlice)
	require.Len(t, documentSymbols, 1)
	require.Len(t, documentSymbols[0].Children, 1)
	assert.Equal(t, "id", documentSymbols[0].Children[0].Name)

	workspaceResult, err := server.Symbols(t.Context(), &protocol.WorkspaceSymbolParams{Query: "id"})
	require.NoError(t, err)
	workspaceSymbols := workspaceResult.(protocol.SymbolInformationSlice)
	require.Len(t, workspaceSymbols, 1)
	assert.Equal(t, "id", workspaceSymbols[0].Name)
	require.NotNil(t, workspaceSymbols[0].ContainerName)
	assert.Equal(t, "User", *workspaceSymbols[0].ContainerName)
}
