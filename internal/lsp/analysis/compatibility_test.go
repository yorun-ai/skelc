package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.yorun.ai/skelc/internal/compiler"
	"go.yorun.ai/skelc/internal/lsp/index"
	"go.yorun.ai/skelc/internal/schema"
)

func TestCompatibilityDiagnosticsUseInMemorySourceAndImpactSeverity(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "contract.skel")
	require.NoError(t, os.WriteFile(path, []byte("domain demo\ndata User { id: int }\n"), 0o600))
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Skel Test")
	runGit(t, root, "config", "user.email", "skel@example.com")
	runGit(t, root, "add", "contract.skel")
	runGit(t, root, "commit", "-m", "baseline")

	documentURI := uri.File(path)
	document := index.Build(documentURI, path, "domain demo\ndata User { id: string }\n", 2)
	sources, paths := SemanticSources(map[uri.URI]*index.Document{documentURI: document})
	analyzer := compiler.NewWorkspaceAnalyzer()
	diagnostics, domains, err := SemanticWorkspace(t.Context(), analyzer, sources, paths)
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	appendCompatibilityDiagnostics(t.Context(), schema.NewSourceDiffer(), diagnostics, domains, sources, paths, CompatibilityOptions{Enabled: true})
	require.Len(t, diagnostics[documentURI], 1)
	result := diagnostics[documentURI][0]
	assert.Equal(t, protocol.String("schema.data.member.type.changed"), result.Code)
	assert.Equal(t, protocol.DiagnosticSeverityWarning, result.Severity)
	assert.Equal(t, protocol.Position{Line: 1, Character: 12}, result.Range.Start)
	assert.Contains(t, result.Message, "[BREAKING]")
}

func TestCompatibilityDiagnosticsPlaceRemovedDeclarationAtDomain(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "contract.skel")
	require.NoError(t, os.WriteFile(path, []byte("domain demo\ndata User {}\n"), 0o600))
	runGit(t, root, "init")
	runGit(t, root, "config", "user.name", "Skel Test")
	runGit(t, root, "config", "user.email", "skel@example.com")
	runGit(t, root, "add", "contract.skel")
	runGit(t, root, "commit", "-m", "baseline")

	documentURI := uri.File(path)
	document := index.Build(documentURI, path, "domain demo\n", 2)
	sources, paths := SemanticSources(map[uri.URI]*index.Document{documentURI: document})
	diagnostics, domains, err := SemanticWorkspace(t.Context(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	appendCompatibilityDiagnostics(t.Context(), schema.NewSourceDiffer(), diagnostics, domains, sources, paths, CompatibilityOptions{Enabled: true})
	require.Len(t, diagnostics[documentURI], 1)
	result := diagnostics[documentURI][0]
	assert.Equal(t, protocol.String("schema.declaration.removed"), result.Code)
	assert.Equal(t, protocol.Position{Line: 0, Character: 7}, result.Range.Start)
}

func TestCompatibilityDiagnosticsReportExplicitBaselineFailure(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "contract.skel")
	baselinePath := filepath.Join(root, "baseline.skel")
	require.NoError(t, os.WriteFile(baselinePath, []byte("domain demo\ndata User { id string }\n"), 0o600))
	documentURI := uri.File(path)
	document := index.Build(documentURI, path, "domain demo\ndata User { id: string }\n", 1)
	sources, paths := SemanticSources(map[uri.URI]*index.Document{documentURI: document})
	diagnostics, domains, err := SemanticWorkspace(t.Context(), compiler.NewWorkspaceAnalyzer(), sources, paths)
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	appendCompatibilityDiagnostics(t.Context(), schema.NewSourceDiffer(), diagnostics, domains, sources, paths,
		CompatibilityOptions{Enabled: true, BaselineSkelIn: baselinePath})
	require.Len(t, diagnostics[documentURI], 1)
	result := diagnostics[documentURI][0]
	assert.Equal(t, protocol.String("schema.baseline"), result.Code)
	assert.Contains(t, result.Message, baselinePath)
}

func runGit(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", directory}, args...)...)
	content, err := command.CombinedOutput()
	require.NoError(t, err, string(content))
}
