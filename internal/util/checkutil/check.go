package checkutil

import (
	"errors"
	"fmt"
	"reflect"
)

// Check panics with a formatted error when condition is false.
func Check(condition bool, message string, args ...any) {
	if !condition {
		Panicf(message, args...)
	}
}

// CheckNot panics with a formatted error when condition is true.
func CheckNot(condition bool, message string, args ...any) {
	if condition {
		Panicf(message, args...)
	}
}

// CheckNotNil panics when value is nil, including typed nil values.
func CheckNotNil(value any, message string, args ...any) {
	if isNil(value) {
		Panicf(message, args...)
	}
}

// CheckNilError panics with a wrapped formatted error when err is non-nil.
func CheckNilError(err error, message string, args ...any) {
	if err != nil {
		panic(fmt.Errorf("%s: %w", fmt.Sprintf(message, args...), err))
	}
}

// CheckFunc panics with the lazily constructed message when condition is false.
func CheckFunc(condition bool, message func() string) {
	if !condition {
		panic(errors.New(message()))
	}
}

// Panicf formats an error and panics with it.
func Panicf(message string, args ...any) {
	panic(fmt.Errorf(message, args...))
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
