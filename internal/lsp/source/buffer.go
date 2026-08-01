// Package source provides UTF-16 position conversion and a tolerant lexical
// view of an in-memory Skel document.
package source

import (
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"go.lsp.dev/protocol"
)

// Buffer is an immutable view of source text.
type Buffer struct {
	content string
}

// Token is a lightweight token used while a document is syntactically
// incomplete. Start and End are byte offsets into the source.
type Token struct {
	Value string
	Start int
	End   int
}

// New creates a source buffer.
func New(content string) Buffer {
	return Buffer{content: content}
}

// String returns the original source text.
func (b Buffer) String() string {
	return b.content
}

// Offset converts an LSP UTF-16 position to a byte offset.
func (b Buffer) Offset(position protocol.Position) int {
	lineStart := 0
	for line := uint32(0); line < position.Line && lineStart < len(b.content); line++ {
		next := strings.IndexByte(b.content[lineStart:], '\n')
		if next < 0 {
			return len(b.content)
		}
		lineStart += next + 1
	}
	offset := lineStart
	units := uint32(0)
	for offset < len(b.content) && b.content[offset] != '\n' && units < position.Character {
		r, size := utf8.DecodeRuneInString(b.content[offset:])
		width := uint32(1)
		if r > 0xffff {
			width = 2
		}
		if units+width > position.Character {
			break
		}
		units += width
		offset += size
	}
	return offset
}

// Position converts a byte offset to an LSP UTF-16 position.
func (b Buffer) Position(offset int) protocol.Position {
	offset = min(max(offset, 0), len(b.content))
	line := strings.Count(b.content[:offset], "\n")
	lineStart := strings.LastIndexByte(b.content[:offset], '\n') + 1
	return protocol.Position{Line: uint32(line), Character: uint32(UTF16Length(b.content[lineStart:offset]))}
}

// Range converts byte offsets to an LSP range.
func (b Buffer) Range(start, end int) protocol.Range {
	return protocol.Range{Start: b.Position(start), End: b.Position(end)}
}

// IdentifierRange converts a one-based parser line and rune column to an LSP
// range. The supplied name determines the range length.
func (b Buffer) IdentifierRange(line, column int, name string) protocol.Range {
	line = max(line-1, 0)
	lineSource := b.line(line)
	byteColumn := 0
	for range max(column-1, 0) {
		if byteColumn >= len(lineSource) {
			break
		}
		_, width := utf8.DecodeRuneInString(lineSource[byteColumn:])
		byteColumn += width
	}
	start := protocol.Position{Line: uint32(line), Character: uint32(UTF16Length(lineSource[:byteColumn]))}
	return protocol.Range{Start: start, End: protocol.Position{Line: start.Line, Character: start.Character + uint32(UTF16Length(name))}}
}

// InNonCode reports whether the position is inside a comment or string.
func (b Buffer) InNonCode(position protocol.Position) bool {
	offset := b.Offset(position)
	const (
		stateCode = iota
		stateLineComment
		stateBlockComment
		stateString
		stateTripleString
	)
	state := stateCode
	for index := 0; index < offset; {
		switch state {
		case stateCode:
			switch {
			case strings.HasPrefix(b.content[index:], "//"):
				state = stateLineComment
				index += 2
			case strings.HasPrefix(b.content[index:], "/*"):
				state = stateBlockComment
				index += 2
			case strings.HasPrefix(b.content[index:], `"""`):
				state = stateTripleString
				index += 3
			case b.content[index] == '"':
				state = stateString
				index++
			default:
				index++
			}
		case stateLineComment:
			if b.content[index] == '\n' {
				state = stateCode
			}
			index++
		case stateBlockComment:
			if strings.HasPrefix(b.content[index:], "*/") {
				state = stateCode
				index += 2
			} else {
				index++
			}
		case stateString:
			if b.content[index] == '\\' {
				index += min(2, offset-index)
			} else {
				if b.content[index] == '"' {
					state = stateCode
				}
				index++
			}
		case stateTripleString:
			if strings.HasPrefix(b.content[index:], `"""`) {
				state = stateCode
				index += 3
			} else {
				index++
			}
		}
	}
	return state != stateCode
}

// IdentifierTokens scans identifiers and the punctuation needed by the
// fallback index while ignoring comments and strings.
func (b Buffer) IdentifierTokens() []Token {
	tokens := make([]Token, 0)
	for offset := 0; offset < len(b.content); {
		switch {
		case strings.HasPrefix(b.content[offset:], "//"):
			if end := strings.IndexByte(b.content[offset:], '\n'); end >= 0 {
				offset += end
			} else {
				return tokens
			}
		case strings.HasPrefix(b.content[offset:], "/*"):
			if end := strings.Index(b.content[offset+2:], "*/"); end >= 0 {
				offset += end + 4
			} else {
				return tokens
			}
		case strings.HasPrefix(b.content[offset:], `"""`):
			if end := strings.Index(b.content[offset+3:], `"""`); end >= 0 {
				offset += end + 6
			} else {
				return tokens
			}
		case b.content[offset] == '"':
			offset++
			for offset < len(b.content) {
				if b.content[offset] == '\\' {
					offset += min(2, len(b.content)-offset)
					continue
				}
				offset++
				if b.content[offset-1] == '"' {
					break
				}
			}
		case b.content[offset] == '.' || b.content[offset] == '{' || b.content[offset] == '}':
			tokens = append(tokens, Token{Value: b.content[offset : offset+1], Start: offset, End: offset + 1})
			offset++
		default:
			r, size := utf8.DecodeRuneInString(b.content[offset:])
			if r == '_' || unicode.IsLetter(r) {
				start := offset
				offset += size
				for offset < len(b.content) {
					r, size = utf8.DecodeRuneInString(b.content[offset:])
					if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
						break
					}
					offset += size
				}
				tokens = append(tokens, Token{Value: b.content[start:offset], Start: start, End: offset})
			} else {
				offset += size
			}
		}
	}
	return tokens
}

// UTF16Length returns the number of UTF-16 code units in value.
func UTF16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

// FirstRune returns the first rune in value.
func FirstRune(value string) rune {
	r, _ := utf8.DecodeRuneInString(value)
	return r
}

func (b Buffer) line(line int) string {
	lines := strings.Split(b.content, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return strings.TrimSuffix(lines[line], "\r")
}
