package lsp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/parser"
)

func TestCodeActionBuildsQuickFixFromSuggestion(t *testing.T) {
	diagnostic := protocol.Diagnostic{
		Range:   protocol.Range{Start: protocol.Position{Line: 2, Character: 8}, End: protocol.Position{Line: 2, Character: 14}},
		Message: protocol.String("incorrect case"),
		Data:    diagnosticSuggestionData(&parser.DiagnosticSuggestion{Message: "replace with userId", Replacement: "userId", Replace: true}),
	}
	documentURI := uri.File("/workspace/user.skel")
	actions, err := newServer().CodeAction(t.Context(), &protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{URI: documentURI},
		Context:      protocol.CodeActionContext{Diagnostics: []protocol.Diagnostic{diagnostic}},
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	action := actions[0].(*protocol.CodeAction)
	require.Equal(t, "replace with userId", action.Title)
	require.Equal(t, "userId", action.Edit.Changes[documentURI][0].NewText)
	require.Equal(t, diagnostic.Range, action.Edit.Changes[documentURI][0].Range)
}
