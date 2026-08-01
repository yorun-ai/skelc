package index

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestIndexDocumentDefinitionsAndReferences(t *testing.T) {
	source := `domain demo.order
import demo.user

// user.Ignored and Ignored must not be indexed.
@desc("user.Ignored")
data Order {
    owner: user.User
    parent: Order?
}
`
	document := Build(uri.File("/workspace/order.skel"), "/workspace/order.skel", source, 1)
	require.Len(t, document.Definitions, 1)
	assert.Equal(t, "demo.order.Order", document.Definitions[0].Key)

	keys := make([]string, 0, len(document.Occurrences))
	for _, occurrence := range document.Occurrences {
		keys = append(keys, occurrence.Key)
	}
	assert.Equal(t, []string{"demo.order.Order", "demo.user.User", "demo.order.Order"}, keys)
}

func TestIndexDocumentUsesUTF16Positions(t *testing.T) {
	source := "domain demo\n@desc(\"𐐀\") data User {}\n"
	document := Build(uri.File("/workspace/user.skel"), "/workspace/user.skel", source, 1)
	require.Len(t, document.Definitions, 1)
	assert.Equal(t, protocol.Position{Line: 1, Character: 17}, document.Definitions[0].Range.Start)
}

func TestIndexDocumentKeepsSyntaxError(t *testing.T) {
	document := Build(uri.File("/workspace/invalid.skel"), "/workspace/invalid.skel", "domain demo\ndata User {", 1)
	require.Len(t, document.ParseDiagnostics, 1)
	assert.Equal(t, "syntax.unexpected-eof", document.ParseDiagnostics[0].Code)
	require.Len(t, document.Definitions, 1)
	assert.Equal(t, "demo.User", document.Definitions[0].Key)
	require.Len(t, document.Occurrences, 1)
	assert.Equal(t, "demo.User", document.Occurrences[0].Key)
}

func TestIndexDocumentBuildsNestedSymbols(t *testing.T) {
	source := `domain demo
service UserService {
    method getUser {
        input {
            userId: int
        }
        output string
    }
}
`
	document := Build(uri.File("/workspace/service.skel"), "/workspace/service.skel", source, 1)
	require.Len(t, document.Symbols, 1)
	service := document.Symbols[0]
	assert.Equal(t, "UserService", service.Name)
	require.Len(t, service.Children, 1)
	method := service.Children[0]
	assert.Equal(t, "getUser", method.Name)
	require.Len(t, method.Children, 1)
	assert.Equal(t, "userId", method.Children[0].Name)
	assert.LessOrEqual(t, comparePosition(method.Children[0].Range.End, method.Range.End), 0)
}
