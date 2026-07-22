package checkutil

import (
	"errors"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckNilErrorWrapsCause(t *testing.T) {
	cause := errors.New("cause")
	defer func() {
		value := recover()
		err, ok := value.(error)
		if !ok || !errors.Is(err, cause) {
			t.Fatalf("expected wrapped cause, got %#v", value)
		}
	}()

	CheckNilError(cause, "operation failed")
}

func TestCheckNotNilRejectsTypedNil(t *testing.T) {
	var value *int
	defer func() {
		if recover() == nil {
			t.Fatal("expected typed nil to panic")
		}
	}()

	CheckNotNil(value, "unexpected nil")
}

func TestFailfCarriesStructuredPosition(t *testing.T) {
	err := Capture(func() {
		Failf("%s %s has an invalid type", lexer.Position{Filename: "/workspace/user.skel", Line: 4, Column: 9}, "field")
	})

	var failure *Failure
	require.ErrorAs(t, err, &failure)
	assert.Equal(t, CodeValidation, failure.Code)
	assert.Equal(t, "/workspace/user.skel", failure.Position.File)
	assert.Equal(t, 4, failure.Position.Line)
	assert.Equal(t, 9, failure.Position.Column)
	assert.Equal(t, "/workspace/user.skel:4:9 field has an invalid type", failure.Message)
}

func TestCapturePreservesWrappedCause(t *testing.T) {
	cause := errors.New("cause")
	err := Capture(func() { CheckNilError(cause, "operation failed") })

	require.Error(t, err)
	assert.ErrorIs(t, err, cause)
}
