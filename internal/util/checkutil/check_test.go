package checkutil

import (
	"errors"
	"testing"
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
