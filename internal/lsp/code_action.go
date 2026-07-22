package lsp

import (
	"context"
	"encoding/json"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/parser"
)

func (s *_Server) CodeAction(_ context.Context, params *protocol.CodeActionParams) ([]protocol.CommandOrCodeAction, error) {
	actions := []protocol.CommandOrCodeAction{}
	for _, diagnostic := range params.Context.Diagnostics {
		if len(diagnostic.Data) == 0 {
			continue
		}
		var suggestion parser.DiagnosticSuggestion
		if err := json.Unmarshal(diagnostic.Data, &suggestion); err != nil || suggestion.Replacement == "" {
			continue
		}
		range_ := diagnostic.Range
		if !suggestion.Replace {
			range_.End = range_.Start
		}
		preferred := true
		kind := protocol.CodeActionKindQuickFix
		actions = append(actions, &protocol.CodeAction{
			Title: suggestion.Message, Kind: &kind, Diagnostics: []protocol.Diagnostic{diagnostic}, IsPreferred: &preferred,
			Edit: &protocol.WorkspaceEdit{Changes: map[uri.URI][]protocol.TextEdit{
				params.TextDocument.URI: {{Range: range_, NewText: suggestion.Replacement}},
			}},
		})
	}
	return actions, nil
}
