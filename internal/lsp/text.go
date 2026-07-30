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

func decoratorPrefixBeforePosition(source string, position protocol.Position) (string, protocol.Range, bool) {
	offset := positionOffset(source, position)
	start := offset
	for start > 0 {
		r, size := utf8.DecodeLastRuneInString(source[:start])
		if r != '_' && !isLetterOrDigit(r) {
			break
		}
		start -= size
	}
	if start == 0 || source[start-1] != '@' {
		return "", protocol.Range{}, false
	}
	return source[start:offset], offsetRange(source, start, offset), true
}

func completionValuesBeforePosition(source string, position protocol.Position) []string {
	offset := positionOffset(source, position)
	lineStart := strings.LastIndexByte(source[:offset], '\n') + 1
	prefix := strings.TrimSpace(source[lineStart:offset])
	fields := strings.Fields(prefix)
	if len(fields) == 0 {
		return nil
	}
	if fields[0] == "pub" {
		fields = fields[1:]
		if len(fields) == 0 {
			return nil
		}
	}
	trailingSpace := offset > lineStart && (source[offset-1] == ' ' || source[offset-1] == '\t')
	if len(fields) >= 2 && fields[0] == "config" {
		switch {
		case len(fields) == 2 && trailingSpace:
			return configLifecycleCompletionValues
		case len(fields) == 3 && isIdentifierValue(fields[2]):
			return configLifecycleCompletionValues
		}
	}
	last := len(fields) - 1
	switch {
	case fields[last] == "via" && trailingSpace:
		return actorViaCompletionValues
	case last > 0 && fields[last-1] == "via" && isIdentifierValue(fields[last]):
		return actorViaCompletionValues
	}
	return nil
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
