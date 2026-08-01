package parser

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSource(t *testing.T) {
	if err := ValidateSource("/tmp/demo.skel", []byte("domain demo.user\n")); err != nil {
		t.Fatal(err)
	}
}

func TestParseSourceReturnsNormalizedSyntaxError(t *testing.T) {
	_, err := ParseSource("/workspace/demo.skel", []byte("domain demo\ndata User {"))
	require.Error(t, err)
	var syntaxError *SyntaxError
	require.True(t, errors.As(err, &syntaxError))
	assert.Equal(t, "/workspace/demo.skel", syntaxError.Position.File)
	assert.True(t, syntaxError.UnexpectedEOF)
	assert.False(t, syntaxError.Finalize)
}

func TestValidateSourceReturnsErrorForInvalidSyntax(t *testing.T) {
	err := ValidateSource("/tmp/demo.skel", []byte("domain {\n"))
	if err == nil || !strings.Contains(err.Error(), "parse /tmp/demo.skel failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseSourceFragmentPreservesOffsets(t *testing.T) {
	result, err := ParseSourceFragment("demo.skel", []byte("data User { id: string }"), 4, 32)
	if err != nil {
		t.Fatal(err)
	}
	if result.Content == nil || len(result.Content.Entries) != 1 {
		t.Fatalf("unexpected parsed content: %+v", result.Content)
	}
	position := result.Content.Entries[0].Data.Name.Pos
	if position.Line != 4 || position.Offset != 37 {
		t.Fatalf("unexpected fragment position: %+v", position)
	}
}
