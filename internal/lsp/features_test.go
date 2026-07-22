package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestServerFormatsValidDocuments(t *testing.T) {
	server := newServer()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, "domain demo\ndata User {\nid: int\n}\n", 1, true)

	edits, err := server.Formatting(t.Context(), &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
	})
	require.NoError(t, err)
	require.Len(t, edits, 1)
	assert.Equal(t, "domain demo\ndata User {\n    id: int\n}\n", edits[0].NewText)
}

func TestServerDoesNotFormatInvalidDocuments(t *testing.T) {
	server := newServer()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, "domain demo\ndata User {", 1, true)

	edits, err := server.Formatting(t.Context(), &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
	})
	require.NoError(t, err)
	assert.Empty(t, edits)
}

func TestServerCompletesKeywordsTypesAndImportedSymbols(t *testing.T) {
	server := newServer()
	userURI := uri.File("/workspace/user.skel")
	orderURI := uri.File("/workspace/order.skel")
	statusURI := uri.File("/workspace/status.skel")
	server.putDocument(userURI, "domain demo.user\ndata User {}\n", 1, true)
	server.putDocument(orderURI, "domain demo.order\nimport demo.user\ndata Order { owner: user. }\n", 1, true)
	server.putDocument(statusURI, "domain demo.order\nenum Status { ACTIVE }\n", 1, true)

	result, err := server.Completion(t.Context(), &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: orderURI},
			Position:     protocol.Position{Line: 2, Character: 25},
		},
	})
	require.NoError(t, err)
	items := result.(protocol.CompletionItemSlice)
	require.Len(t, items, 1)
	assert.Equal(t, "User", items[0].Label)

	result, err = server.Completion(t.Context(), &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: orderURI},
			Position:     protocol.Position{Line: 2, Character: 13},
		},
	})
	require.NoError(t, err)
	items = result.(protocol.CompletionItemSlice)
	assert.True(t, hasCompletion(items, "service"))
	assert.True(t, hasCompletion(items, "string"))
	assert.True(t, hasCompletion(items, "Order"))
	assert.True(t, hasCompletion(items, "Status"))
	assert.True(t, hasCompletion(items, "user"))
}

func TestServerHoverShowsQualifiedDeclaration(t *testing.T) {
	server := newServer()
	userURI := uri.File("/workspace/user.skel")
	orderURI := uri.File("/workspace/order.skel")
	server.putDocument(userURI, "domain demo.user\n@desc(\"Account data\")\ndata User {}\n", 1, true)
	server.putDocument(orderURI, "domain demo.order\nimport demo.user\ndata Order { owner: user.User }\n", 1, true)

	hover, err := server.Hover(t.Context(), &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: orderURI},
			Position:     protocol.Position{Line: 2, Character: 27},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover)
	content := hover.Contents.(*protocol.MarkupContent)
	assert.Contains(t, content.Value, "data demo.user.User")
	assert.Contains(t, content.Value, "Account data")
}

func TestServerSuppressesCompletionInCommentsAndHoversBuiltInTypes(t *testing.T) {
	server := newServer()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, "domain demo\n// string\ndata User { id: int }\n", 1, true)

	result, err := server.Completion(t.Context(), &protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
			Position:     protocol.Position{Line: 1, Character: 5},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, result.(protocol.CompletionItemSlice))

	hover, err := server.Hover(t.Context(), &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
			Position:     protocol.Position{Line: 2, Character: 17},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, hover)
	assert.Contains(t, hover.Contents.(*protocol.MarkupContent).Value, "built-in type int")
}

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

func TestServerRenamesDeclarationsAndReferences(t *testing.T) {
	server := newServer()
	userURI := uri.File("/workspace/user.skel")
	orderURI := uri.File("/workspace/order.skel")
	server.putDocument(userURI, "domain demo.user\ndata User {}\n", 1, true)
	server.putDocument(orderURI, "domain demo.order\nimport demo.user\ndata Order { owner: user.User }\n", 1, true)

	edit, err := server.Rename(t.Context(), &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: userURI},
			Position:     protocol.Position{Line: 1, Character: 6},
		},
		NewName: "Account",
	})
	require.NoError(t, err)
	require.Len(t, edit.Changes[userURI], 1)
	require.Len(t, edit.Changes[orderURI], 1)
	assert.Equal(t, "Account", edit.Changes[userURI][0].NewText)
	assert.Equal(t, "Account", edit.Changes[orderURI][0].NewText)
}

func TestServerRejectsInvalidRename(t *testing.T) {
	server := newServer()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, "domain demo\ndata User {}\n", 1, true)

	_, err := server.Rename(t.Context(), &protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
			Position:     protocol.Position{Line: 1, Character: 6},
		},
		NewName: "not valid",
	})
	assert.ErrorContains(t, err, "invalid Skel identifier")
}

func TestServerDoesNotRenameUnresolvedReferences(t *testing.T) {
	server := newServer()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, "domain demo\ndata User { missing: Missing }\n", 1, true)

	prepared, err := server.PrepareRename(t.Context(), &protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
			Position:     protocol.Position{Line: 1, Character: 23},
		},
	})
	require.NoError(t, err)
	assert.Nil(t, prepared)
}

func hasCompletion(items protocol.CompletionItemSlice, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}
