package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

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

func hasCompletion(items protocol.CompletionItemSlice, label string) bool {
	for _, item := range items {
		if item.Label == label {
			return true
		}
	}
	return false
}
