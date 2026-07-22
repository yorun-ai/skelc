package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

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
