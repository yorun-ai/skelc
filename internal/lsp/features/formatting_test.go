package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestServiceFormatsValidDocuments(t *testing.T) {
	server := newFixture()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, "domain demo\ndata User {\nid: int\n}\n", 1, true)

	edits, err := server.Formatting(t.Context(), &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
	})
	require.NoError(t, err)
	require.Len(t, edits, 1)
	assert.Equal(t, "domain demo\n\ndata User {\n    id: int\n}\n", edits[0].NewText)
}

func TestServiceDoesNotFormatInvalidDocuments(t *testing.T) {
	server := newFixture()
	documentURI := uri.File("/workspace/user.skel")
	server.putDocument(documentURI, "domain demo\ndata User {", 1, true)

	edits, err := server.Formatting(t.Context(), &protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
	})
	require.NoError(t, err)
	assert.Empty(t, edits)
}
