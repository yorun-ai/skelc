package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeWorkspaceValidatesCrossDomainTypes(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{
		{Path: "/workspace/user.skel", Content: []byte("domain demo.user\npub data User { id: int }\n")},
		{Path: "/workspace/order.skel", Content: []byte("domain demo.order\nimport demo.user\ndata Order { owner: user.Missing }\n")},
	})

	require.Len(t, diagnostics, 1)
	assert.Equal(t, DiagnosticCodeSemantic, diagnostics[0].Code)
	assert.Equal(t, "/workspace/order.skel", diagnostics[0].Position.File)
	assert.Equal(t, 3, diagnostics[0].Position.Line)
	assert.Contains(t, diagnostics[0].Message, "definition of user.Missing not found")
}

func TestAnalyzeWorkspaceMergesSameDomainFiles(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{
		{Path: "/workspace/data.skel", Content: []byte("domain demo.user\ndata User { id: int }\n")},
		{Path: "/workspace/service.skel", Content: []byte("domain demo.user\nservice UserService { method get { output User } }\n")},
	})

	assert.Empty(t, diagnostics)
}

func TestAnalyzeWorkspaceReturnsStructuredDuplicatePosition(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{
		{Path: "/workspace/a.skel", Content: []byte("domain demo\ndata User {}\n")},
		{Path: "/workspace/b.skel", Content: []byte("domain demo\ndata User {}\n")},
	})

	require.Len(t, diagnostics, 1)
	assert.Equal(t, DiagnosticCodeSemantic, diagnostics[0].Code)
	assert.Equal(t, "/workspace/b.skel", diagnostics[0].Position.File)
	assert.Equal(t, 2, diagnostics[0].Position.Line)
	assert.NotContains(t, diagnostics[0].Message, "/workspace/b.skel:2:6")
}

func TestAnalyzeWorkspaceSuppressesDependentCascade(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{
		{Path: "/workspace/user.skel", Domain: "demo.user", Content: []byte("domain demo.user\ndata User {")},
		{Path: "/workspace/order.skel", Content: []byte("domain demo.order\nimport demo.user\ndata Order { owner: user.User }\n")},
	})

	require.Len(t, diagnostics, 1)
	assert.Equal(t, DiagnosticCodeSyntax, diagnostics[0].Code)
	assert.Equal(t, "/workspace/user.skel", diagnostics[0].Position.File)
}

func TestAnalyzeWorkspaceReportsMissingImport(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{{
		Path: "/workspace/order.skel", Content: []byte("domain demo.order\nimport demo.user\ndata Order {}\n"),
	}})

	require.Len(t, diagnostics, 1)
	assert.Equal(t, DiagnosticCodeImport, diagnostics[0].Code)
	assert.Equal(t, 2, diagnostics[0].Position.Line)
}
