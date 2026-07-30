package lsp

import (
	"context"
	"net"
	"os"
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
	require.NotNil(t, result.Capabilities.Workspace)
	require.NotNil(t, result.Capabilities.Workspace.WorkspaceFolders)
	require.NotNil(t, result.Capabilities.Workspace.WorkspaceFolders.Supported)
	assert.True(t, *result.Capabilities.Workspace.WorkspaceFolders.Supported)
	assert.Equal(t, protocol.Boolean(true), result.Capabilities.Workspace.WorkspaceFolders.ChangeNotifications)
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
		return params.URI == orderURI && len(params.Diagnostics) == 1 && params.Diagnostics[0].Code == protocol.String("semantic.reference")
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

func TestServerPreservesRemoteWorkspaceURIs(t *testing.T) {
	rootPath := t.TempDir()
	userPath := filepath.Join(rootPath, "user.skel")
	require.NoError(t, os.WriteFile(userPath, []byte("domain demo.user\ndata User {}\n"), 0o600))
	rootURI, err := uri.From(uri.Components{
		Scheme: "vscode-remote", Authority: "ssh-remote+test", Path: filepath.ToSlash(rootPath),
	})
	require.NoError(t, err)
	userURI, err := uri.JoinPath(rootURI, "user.skel")
	require.NoError(t, err)

	server := newServer()
	_, err = server.Initialize(t.Context(), &protocol.InitializeParams{
		WorkspaceFoldersInitializeParams: protocol.WorkspaceFoldersInitializeParams{
			WorkspaceFolders: protocol.NewNullable([]protocol.WorkspaceFolder{{URI: rootURI, Name: "remote"}}),
		},
		Capabilities: protocol.ClientCapabilities{},
	})
	require.NoError(t, err)
	require.Contains(t, server.documents, userURI)
	assert.NotContains(t, server.documents, uri.File(userPath))

	require.NoError(t, server.DidOpen(t.Context(), &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI: userURI, LanguageID: "skel", Version: 1, Text: "domain demo.user\ndata Profile {}\n",
		},
	}))
	require.Len(t, server.documents, 1)
	assert.Equal(t, int32(1), server.documents[userURI].Version)
	assert.Equal(t, "Profile", server.documents[userURI].Definitions[0].Name)
}

func TestServerUpdatesDynamicWorkspaceFolders(t *testing.T) {
	firstPath := t.TempDir()
	secondPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(firstPath, "first.skel"), []byte("domain demo.first\ndata First {}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(secondPath, "second.skel"), []byte("domain demo.second\ndata Second {}\n"), 0o600))
	firstURI := uri.File(firstPath)
	secondURI := uri.File(secondPath)
	firstDocumentURI := uri.File(filepath.Join(firstPath, "first.skel"))
	secondDocumentURI := uri.File(filepath.Join(secondPath, "second.skel"))

	server := newServer()
	_, err := server.Initialize(t.Context(), &protocol.InitializeParams{
		WorkspaceFoldersInitializeParams: protocol.WorkspaceFoldersInitializeParams{
			WorkspaceFolders: protocol.NewNullable([]protocol.WorkspaceFolder{{URI: firstURI, Name: "first"}}),
		},
		Capabilities: protocol.ClientCapabilities{},
	})
	require.NoError(t, err)
	require.Contains(t, server.documents, firstDocumentURI)

	require.NoError(t, server.DidChangeWorkspaceFolders(t.Context(), &protocol.DidChangeWorkspaceFoldersParams{
		Event: protocol.WorkspaceFoldersChangeEvent{
			Added:   []protocol.WorkspaceFolder{{URI: secondURI, Name: "second"}},
			Removed: []protocol.WorkspaceFolder{{URI: firstURI, Name: "first"}},
		},
	}))
	assert.NotContains(t, server.documents, firstDocumentURI)
	assert.Contains(t, server.documents, secondDocumentURI)
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
