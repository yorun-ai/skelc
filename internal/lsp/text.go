package lsp

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"go.lsp.dev/protocol"
)

func qualifierBeforePosition(source string, position protocol.Position) string {
	offset := positionOffset(source, position)
	start := offset
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(source[:start])
		if r != '_' && !isLetterOrDigit(r) {
			break
		}
		start -= size
	}
	if start == 0 || source[start-1] != '.' {
		return ""
	}
	end := start - 1
	start = end
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(source[:start])
		if r != '_' && !isLetterOrDigit(r) {
			break
		}
		start -= size
	}
	return source[start:end]
}

func positionOffset(source string, position protocol.Position) int {
	lineStart := 0
	for line := uint32(0); line < position.Line && lineStart < len(source); line++ {
		next := strings.IndexByte(source[lineStart:], '\n')
		if next < 0 {
			return len(source)
		}
		lineStart += next + 1
	}
	offset := lineStart
	units := uint32(0)
	for offset < len(source) && source[offset] != '\n' && units < position.Character {
		r, size := utf8.DecodeRuneInString(source[offset:])
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

func positionInNonCode(source string, position protocol.Position) bool {
	offset := positionOffset(source, position)
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
			case strings.HasPrefix(source[index:], "//"):
				state = stateLineComment
				index += 2
			case strings.HasPrefix(source[index:], "/*"):
				state = stateBlockComment
				index += 2
			case strings.HasPrefix(source[index:], `"""`):
				state = stateTripleString
				index += 3
			case source[index] == '"':
				state = stateString
				index++
			default:
				index++
			}
		case stateLineComment:
			if source[index] == '\n' {
				state = stateCode
			}
			index++
		case stateBlockComment:
			if strings.HasPrefix(source[index:], "*/") {
				state = stateCode
				index += 2
			} else {
				index++
			}
		case stateString:
			if source[index] == '\\' {
				index += min(2, offset-index)
			} else {
				if source[index] == '"' {
					state = stateCode
				}
				index++
			}
		case stateTripleString:
			if strings.HasPrefix(source[index:], `"""`) {
				state = stateCode
				index += 3
			} else {
				index++
			}
		}
	}
	return state != stateCode
}

func isLetterOrDigit(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
