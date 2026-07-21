package lsp

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

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
