package features

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestServiceDefinitionAndReferencesAcrossDomains(t *testing.T) {
	server := newFixture()
	userURI := uri.File("/workspace/user.skel")
	orderURI := uri.File("/workspace/order.skel")
	server.putDocument(userURI, "domain demo.user\ndata User {}\n", 1, true)
	server.putDocument(orderURI, "domain demo.order\nimport demo.user\ndata Order { owner: user.User }\n", 1, true)

	definition, err := server.Definition(t.Context(), &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: orderURI},
			Position:     protocol.Position{Line: 2, Character: 25},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, protocol.LocationSlice{{
		URI: userURI,
		Range: protocol.Range{
			Start: protocol.Position{Line: 1, Character: 5},
			End:   protocol.Position{Line: 1, Character: 9},
		},
	}}, definition)

	references, err := server.References(t.Context(), &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: userURI},
			Position:     protocol.Position{Line: 1, Character: 6},
		},
		Context: protocol.ReferenceContext{IncludeDeclaration: true},
	})
	require.NoError(t, err)
	require.Len(t, references, 2)
	assert.Equal(t, orderURI, references[0].URI)
	assert.Equal(t, userURI, references[1].URI)

	references, err = server.References(t.Context(), &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: userURI},
			Position:     protocol.Position{Line: 1, Character: 6},
		},
	})
	require.NoError(t, err)
	require.Len(t, references, 1)
	assert.Equal(t, orderURI, references[0].URI)
}
