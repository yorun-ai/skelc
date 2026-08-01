package features

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

func isLetterOrDigit(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}
