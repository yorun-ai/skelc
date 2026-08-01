package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.lsp.dev/protocol"
)

func TestBufferConvertsUTF16Positions(t *testing.T) {
	buffer := New("data 𐐀User {}\r\n")
	position := buffer.Position(len("data 𐐀"))
	assert.Equal(t, protocol.Position{Line: 0, Character: 7}, position)
	assert.Equal(t, len("data 𐐀"), buffer.Offset(position))
	assert.Equal(t, protocol.Range{
		Start: protocol.Position{Line: 0, Character: 7},
		End:   protocol.Position{Line: 0, Character: 11},
	}, buffer.IdentifierRange(1, 7, "User"))
}

func TestBufferScansIdentifiersOutsideCommentsAndStrings(t *testing.T) {
	buffer := New("data User { // Ignored\n value: Other @desc(\"Hidden\")\n}\n")
	tokens := buffer.IdentifierTokens()
	values := make([]string, 0, len(tokens))
	for _, token := range tokens {
		values = append(values, token.Value)
	}
	assert.Equal(t, []string{"data", "User", "{", "value", "Other", "desc", "}"}, values)
	assert.True(t, buffer.InNonCode(protocol.Position{Line: 0, Character: 20}))
	assert.True(t, buffer.InNonCode(protocol.Position{Line: 1, Character: 22}))
	assert.False(t, buffer.InNonCode(protocol.Position{Line: 1, Character: 2}))
}
