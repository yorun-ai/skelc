package checkutil

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/model"
)

const CodeValidation = "validation"

// Failure is the structured validation failure used to abort a deeply nested
// compiler operation. Error preserves the existing human-readable message,
// while Position and Code let tool integrations avoid parsing that message.
type Failure struct {
	Code     string
	Position model.Position
	Message  string
	Cause    error
	Related  []RelatedLocation
}

type RelatedLocation struct {
	Position model.Position
	Message  string
}

func (f *Failure) Error() string { return f.Message }

func (f *Failure) Unwrap() error { return f.Cause }

// SourcePosition returns the source position associated with the failure.
func (f *Failure) SourcePosition() model.Position { return f.Position }

// Check aborts with a structured failure when condition is false.
func Check(condition bool, message string, args ...any) {
	if !condition {
		Failf(message, args...)
	}
}

// CheckNot aborts with a structured failure when condition is true.
func CheckNot(condition bool, message string, args ...any) {
	if condition {
		Failf(message, args...)
	}
}

// CheckNotNil aborts when value is nil, including typed nil values.
func CheckNotNil(value any, message string, args ...any) {
	if isNil(value) {
		Failf(message, args...)
	}
}

// CheckNilError aborts with a structured failure that wraps err.
func CheckNilError(err error, message string, args ...any) {
	if err == nil {
		return
	}
	prefix := fmt.Sprintf(message, args...)
	panic(NewFailureWithCause(err, prefix+": "+err.Error(), args...))
}

// CheckFuncAt aborts with a lazily constructed structured failure at position.
func CheckFuncAt(position any, condition bool, message func() string) {
	if !condition {
		panic(&Failure{
			Code: CodeValidation, Position: positionFromArgs([]any{position}), Message: message(),
		})
	}
}

// Failf formats a message, infers a source position from its arguments when
// possible, and aborts with a structured failure.
func Failf(message string, args ...any) {
	panic(NewFailuref(message, args...))
}

// NewFailuref constructs a structured validation failure without aborting the
// current operation.
func NewFailuref(message string, args ...any) *Failure {
	return &Failure{
		Code: CodeValidation, Position: positionFromArgs(args), Message: fmt.Sprintf(message, args...),
	}
}

// NewFailureWithCause constructs a structured validation failure that wraps a cause.
func NewFailureWithCause(cause error, message string, args ...any) *Failure {
	return &Failure{
		Code: CodeValidation, Position: positionFromArgs(args), Message: message, Cause: cause,
	}
}

// Position returns the structured source position carried by err.
func Position(err error) (model.Position, bool) {
	var positioned interface{ SourcePosition() model.Position }
	if !errors.As(err, &positioned) {
		return model.Position{}, false
	}
	position := positioned.SourcePosition()
	return position, position.Line > 0
}

func positionFromArgs(args []any) model.Position {
	for _, arg := range args {
		switch position := arg.(type) {
		case model.Position:
			return position
		case *model.Position:
			if position != nil {
				return *position
			}
		case lexer.Position:
			return model.Position{File: position.Filename, Line: position.Line, Column: position.Column}
		case *lexer.Position:
			if position != nil {
				return model.Position{File: position.Filename, Line: position.Line, Column: position.Column}
			}
		}
	}
	return model.Position{}
}

func isNil(value any) bool {
	reflected := reflect.ValueOf(value)
	if !reflected.IsValid() {
		return true
	}

	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
