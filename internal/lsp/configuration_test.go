package lsp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

func TestDecodeChangedSettingsMergesPartialUpdates(t *testing.T) {
	fallback := _SchemaCompatibilitySettings{
		Diagnostics: true, IncludeCompatible: true, CodeLens: true, Baseline: "../baseline",
	}
	for _, test := range []struct {
		name     string
		settings any
	}{
		{name: "direct", settings: map[string]any{"schemaCompatibility": map[string]any{"codeLens": false}}},
		{name: "skelc envelope", settings: map[string]any{"skelc": map[string]any{
			"schemaCompatibility": map[string]any{"codeLens": false},
		}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			content, err := json.Marshal(test.settings)
			require.NoError(t, err)
			assert.Equal(t, _SchemaCompatibilitySettings{
				Diagnostics: true, IncludeCompatible: true, CodeLens: false, Baseline: "../baseline",
			}, decodeChangedSettings(content, fallback))
		})
	}
}

func TestDidChangeConfigurationClearsStaleCompatibilityDiagnostics(t *testing.T) {
	server := newServer()
	t.Cleanup(server.stopSemanticAnalysis)
	documentURI := uri.File("/workspace/domain.skel")
	client := &recordingClient{diagnostics: make(chan *protocol.PublishDiagnosticsParams, 1)}
	server.client = client
	server.schemaCompatibility = _SchemaCompatibilitySettings{Diagnostics: true, CodeLens: true}
	server.semantic[documentURI] = []protocol.Diagnostic{{Code: protocol.String("schema.declaration.removed")}}
	settings, err := json.Marshal(map[string]any{
		"schemaCompatibility": map[string]any{"diagnostics": false},
	})
	require.NoError(t, err)

	require.NoError(t, server.DidChangeConfiguration(t.Context(), &protocol.DidChangeConfigurationParams{Settings: settings}))
	assert.False(t, server.schemaCompatibility.Diagnostics)
	assert.True(t, server.schemaCompatibility.CodeLens)
	assert.Empty(t, server.semantic)
	select {
	case published := <-client.diagnostics:
		assert.Equal(t, documentURI, published.URI)
		assert.Empty(t, published.Diagnostics)
	default:
		t.Fatal("expected stale compatibility diagnostics to be cleared")
	}
}
