package lsp

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type recordingClient struct {
	protocol.UnimplementedClient
	diagnostics chan *protocol.PublishDiagnosticsParams
}

func (c *recordingClient) PublishDiagnostics(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
	c.diagnostics <- params
	return nil
}

func TestServeLifecycle(t *testing.T) {
	serverStream, clientStream := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- Serve(t.Context(), serverStream, serverStream)
	}()

	_, connection, server := protocol.NewClient(
		t.Context(), protocol.UnimplementedClient{}, jsonrpc2.NewStream(clientStream),
	)
	t.Cleanup(func() { _ = connection.Close() })

	root := uri.File(t.TempDir())
	result, err := server.Initialize(t.Context(), &protocol.InitializeParams{
		RootURI: &root, Capabilities: protocol.ClientCapabilities{},
	})
	require.NoError(t, err)
	assert.Equal(t, protocol.PositionEncodingKindUTF16, result.Capabilities.PositionEncoding)
	require.NotNil(t, result.Capabilities.CompletionProvider)
	assert.Equal(t, []string{"."}, result.Capabilities.CompletionProvider.TriggerCharacters)
	assert.Equal(t, protocol.Boolean(true), result.Capabilities.HoverProvider)
	assert.Equal(t, protocol.Boolean(true), result.Capabilities.WorkspaceSymbolProvider)
	assert.Equal(t, protocol.Boolean(true), result.Capabilities.DocumentFormattingProvider)
	rename, ok := result.Capabilities.RenameProvider.(*protocol.RenameOptions)
	require.True(t, ok)
	require.NotNil(t, rename.PrepareProvider)
	assert.True(t, *rename.PrepareProvider)
	require.NoError(t, server.Initialized(t.Context(), &protocol.InitializedParams{}))
	require.NoError(t, server.Shutdown(t.Context()))
	require.NoError(t, server.Exit(t.Context()))

	select {
	case err := <-serverDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("language server did not exit after the exit notification")
	}
}

func TestServePublishesAndInvalidatesSemanticDiagnostics(t *testing.T) {
	serverStream, clientStream := net.Pipe()
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- Serve(t.Context(), serverStream, serverStream)
	}()

	client := &recordingClient{diagnostics: make(chan *protocol.PublishDiagnosticsParams, 16)}
	_, connection, server := protocol.NewClient(t.Context(), client, jsonrpc2.NewStream(clientStream))
	t.Cleanup(func() { _ = connection.Close() })

	rootPath := t.TempDir()
	root := uri.File(rootPath)
	_, err := server.Initialize(t.Context(), &protocol.InitializeParams{
		RootURI: &root, Capabilities: protocol.ClientCapabilities{},
	})
	require.NoError(t, err)
	require.NoError(t, server.Initialized(t.Context(), &protocol.InitializedParams{}))

	userURI := uri.File(filepath.Join(rootPath, "user.skel"))
	orderURI := uri.File(filepath.Join(rootPath, "order.skel"))
	require.NoError(t, server.DidOpen(t.Context(), &protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{
		URI: userURI, LanguageID: "skel", Version: 1, Text: "domain demo.user\ndata User {}\n",
	}}))
	require.NoError(t, server.DidOpen(t.Context(), &protocol.DidOpenTextDocumentParams{TextDocument: protocol.TextDocumentItem{
		URI: orderURI, LanguageID: "skel", Version: 1,
		Text: "domain demo.order\nimport demo.user\ndata Order { owner: user.Missing }\n",
	}}))

	diagnostic := waitForDiagnostics(t, client.diagnostics, func(params *protocol.PublishDiagnosticsParams) bool {
		return params.URI == orderURI && len(params.Diagnostics) == 1 && params.Diagnostics[0].Code == protocol.String("semantic")
	})
	assert.Equal(t, protocol.NewOptional(int32(1)), diagnostic.Version)

	require.NoError(t, server.DidChange(t.Context(), &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: orderURI}, Version: 2,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{&protocol.TextDocumentContentChangeWholeDocument{
			Text: "domain demo.order\nimport demo.user\ndata Order { owner: user.User }\n",
		}},
	}))
	waitForDiagnostics(t, client.diagnostics, func(params *protocol.PublishDiagnosticsParams) bool {
		return params.URI == orderURI && params.Version == protocol.NewOptional(int32(2)) && len(params.Diagnostics) == 0
	})

	require.NoError(t, server.Shutdown(t.Context()))
	require.NoError(t, server.Exit(t.Context()))
	select {
	case err := <-serverDone:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("language server did not exit after the exit notification")
	}
}

func waitForDiagnostics(
	t *testing.T,
	diagnostics <-chan *protocol.PublishDiagnosticsParams,
	accept func(*protocol.PublishDiagnosticsParams) bool,
) *protocol.PublishDiagnosticsParams {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case params := <-diagnostics:
			if accept(params) {
				return params
			}
		case <-timer.C:
			t.Fatal("timed out waiting for matching diagnostics")
		}
	}
}
