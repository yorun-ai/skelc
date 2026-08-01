package grammar

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// UnexpectedEOFError identifies incomplete grammar input without requiring
// callers to inspect its human-readable message.
type UnexpectedEOFError struct {
	Pos     lexer.Position
	Context string
}

func (e *UnexpectedEOFError) Error() string { return participle.FormatError(e) }

func (e *UnexpectedEOFError) Message() string {
	return fmt.Sprintf("unexpected EOF in %s", e.Context)
}

func (e *UnexpectedEOFError) Position() lexer.Position { return e.Pos }
