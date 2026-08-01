package features

import (
	"context"
	"encoding/json"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/compiler"
	"go.yorun.ai/skelc/internal/lsp/workspace"
)

type testFixture struct {
	workspace      *workspace.Store
	snippetSupport bool
}

func newFixture() *testFixture {
	return &testFixture{workspace: workspace.New()}
}

func (f *testFixture) putDocument(documentURI uri.URI, content string, version int32, open bool) {
	f.workspace.Put(documentURI, content, version, open)
}

func (f *testFixture) service() Service {
	return Service{Snapshot: f.workspace.Snapshot(), SnippetSupport: f.snippetSupport}
}

func (f *testFixture) Completion(ctx context.Context, params *protocol.CompletionParams) (protocol.CompletionResult, error) {
	service := f.service()
	return service.Completion(ctx, params)
}

func (f *testFixture) Hover(ctx context.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	service := f.service()
	return service.Hover(ctx, params)
}

func (f *testFixture) Definition(ctx context.Context, params *protocol.DefinitionParams) (protocol.DefinitionResult, error) {
	service := f.service()
	return service.Definition(ctx, params)
}

func (f *testFixture) References(ctx context.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	service := f.service()
	return service.References(ctx, params)
}

func (f *testFixture) DocumentSymbol(ctx context.Context, params *protocol.DocumentSymbolParams) (protocol.DocumentSymbolResult, error) {
	service := f.service()
	return service.DocumentSymbol(ctx, params)
}

func (f *testFixture) Symbols(ctx context.Context, params *protocol.WorkspaceSymbolParams) (protocol.WorkspaceSymbolResult, error) {
	service := f.service()
	return service.Symbols(ctx, params)
}

func (f *testFixture) PrepareRename(ctx context.Context, params *protocol.PrepareRenameParams) (protocol.PrepareRenameResult, error) {
	service := f.service()
	return service.PrepareRename(ctx, params)
}

func (f *testFixture) Rename(ctx context.Context, params *protocol.RenameParams) (*protocol.WorkspaceEdit, error) {
	service := f.service()
	return service.Rename(ctx, params)
}

func (f *testFixture) Formatting(ctx context.Context, params *protocol.DocumentFormattingParams) ([]protocol.TextEdit, error) {
	service := f.service()
	return service.Formatting(ctx, params)
}

func (f *testFixture) CodeAction(ctx context.Context, params *protocol.CodeActionParams) ([]protocol.CommandOrCodeAction, error) {
	service := f.service()
	return service.CodeAction(ctx, params)
}

func diagnosticSuggestionData(suggestion *compiler.DiagnosticSuggestion) protocol.LSPAny {
	content, err := json.Marshal(suggestion)
	if err != nil {
		return nil
	}
	return protocol.LSPAny(content)
}
