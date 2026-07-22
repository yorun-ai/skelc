package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yorun.ai/skelc/internal/parser/analyzer"
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

func TestAnalyzeWorkspaceCollectsMultipleSemanticDiagnosticsInOneDomain(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{{
		Path: "/workspace/data.skel",
		Content: []byte(`domain demo
data User { missing: MissingUser }
data Order { missing: MissingOrder }
`),
	}})

	require.Len(t, diagnostics, 2)
	assert.Equal(t, []int{2, 3}, []int{diagnostics[0].Position.Line, diagnostics[1].Position.Line})
	assert.Contains(t, diagnostics[0].Message, "definition of MissingUser not found")
	assert.Contains(t, diagnostics[1].Message, "definition of MissingOrder not found")
}

func TestAnalyzeWorkspaceSuppressesInvalidDeclarationCascades(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{{
		Path: "/workspace/data.skel",
		Content: []byte(`domain demo
data User {
    id: int
    id: string
}
data Order { owner: User }
data Product { missing: MissingProduct }
`),
	}})

	require.Len(t, diagnostics, 2)
	assert.Contains(t, diagnostics[0].Message, "duplicated DataMember id")
	assert.Contains(t, diagnostics[1].Message, "definition of MissingProduct not found")
	for _, diagnostic := range diagnostics {
		assert.NotContains(t, diagnostic.Message, "definition of User not found")
	}
}

func TestAnalyzeWorkspaceCollectsMultipleMemberDiagnostics(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{{
		Path: "/workspace/data.skel",
		Content: []byte(`domain demo
data User {
    first: MissingFirst
    second: MissingSecond
}
`),
	}})

	require.Len(t, diagnostics, 2)
	assert.Contains(t, diagnostics[0].Message, "definition of MissingFirst not found")
	assert.Contains(t, diagnostics[1].Message, "definition of MissingSecond not found")
}

func TestAnalyzeWorkspaceLimitsDiagnosticsPerDomain(t *testing.T) {
	var source strings.Builder
	source.WriteString("domain demo\n")
	for index := 0; index < analyzer.MaxDiagnosticsPerDomain+10; index++ {
		_, _ = fmt.Fprintf(&source, "data Type%d { missing: Missing%d }\n", index, index)
	}

	diagnostics := AnalyzeWorkspace([]Source{{Path: "/workspace/data.skel", Content: []byte(source.String())}})

	assert.Len(t, diagnostics, analyzer.MaxDiagnosticsPerDomain)
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

func TestAnalyzeWorkspaceReportsMultipleMissingImports(t *testing.T) {
	diagnostics := AnalyzeWorkspace([]Source{{
		Path:    "/workspace/order.skel",
		Content: []byte("domain demo.order\nimport demo.user\nimport demo.product\ndata Order {}\n"),
	}})

	require.Len(t, diagnostics, 2)
	assert.Equal(t, []int{2, 3}, []int{diagnostics[0].Position.Line, diagnostics[1].Position.Line})
	assert.Equal(t, DiagnosticCodeImport, diagnostics[0].Code)
	assert.Equal(t, DiagnosticCodeImport, diagnostics[1].Code)
}
