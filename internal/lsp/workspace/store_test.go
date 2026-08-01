package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/uri"
)

func TestStorePreservesRemoteWorkspaceURIsAndOpenContents(t *testing.T) {
	rootPath := t.TempDir()
	userPath := filepath.Join(rootPath, "user.skel")
	require.NoError(t, os.WriteFile(userPath, []byte("domain demo.user\ndata User {}\n"), 0o600))
	rootURI, err := uri.From(uri.Components{
		Scheme: "vscode-remote", Authority: "ssh-remote+test", Path: filepath.ToSlash(rootPath),
	})
	require.NoError(t, err)
	userURI, err := uri.JoinPath(rootURI, "user.skel")
	require.NoError(t, err)

	store := New()
	store.AddRoot(rootURI)
	snapshot := store.Snapshot()
	require.NotNil(t, snapshot.Document(userURI))
	assert.Nil(t, snapshot.Document(uri.File(userPath)))

	store.Put(userURI, "domain demo.user\ndata Profile {}\n", 1, true)
	snapshot = store.Snapshot()
	require.Len(t, snapshot.Documents(), 1)
	assert.Equal(t, int32(1), snapshot.Document(userURI).Version)
	assert.Equal(t, "Profile", snapshot.Document(userURI).Definitions[0].Name)
}

func TestStoreAddsAndRemovesWorkspaceRoots(t *testing.T) {
	firstPath := t.TempDir()
	secondPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(firstPath, "first.skel"), []byte("domain demo.first\ndata First {}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(secondPath, "second.skel"), []byte("domain demo.second\ndata Second {}\n"), 0o600))
	firstURI := uri.File(firstPath)
	secondURI := uri.File(secondPath)
	firstDocumentURI := uri.File(filepath.Join(firstPath, "first.skel"))
	secondDocumentURI := uri.File(filepath.Join(secondPath, "second.skel"))

	store := New()
	store.AddRoot(firstURI)
	require.NotNil(t, store.Snapshot().Document(firstDocumentURI))

	removed := store.RemoveRoot(firstURI)
	store.AddRoot(secondURI)
	assert.Equal(t, []uri.URI{firstDocumentURI}, removed)
	snapshot := store.Snapshot()
	assert.Nil(t, snapshot.Document(firstDocumentURI))
	assert.NotNil(t, snapshot.Document(secondDocumentURI))
}
