package lsp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

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
