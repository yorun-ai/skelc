package compiler

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"go.yorun.ai/skelc/model"
)

func sourceLineOffsets(source []byte, line int) (int, int, bool) {
	if line <= 0 {
		return 0, 0, false
	}
	start := 0
	for current := 1; current < line; current++ {
		index := bytes.IndexByte(source[start:], '\n')
		if index < 0 {
			return 0, 0, false
		}
		start += index + 1
	}
	if start > len(source) {
		return 0, 0, false
	}
	end := len(source)
	if index := bytes.IndexByte(source[start:], '\n'); index >= 0 {
		end = start + index
	}
	if end > start && source[end-1] == '\r' {
		end--
	}
	return start, end, true
}

func sourceRangeAt(start model.Position, source []byte) SourceRange {
	end := start
	if start.Line <= 0 || start.Column <= 0 {
		return SourceRange{Start: start, End: end}
	}
	lineStart, lineEnd, ok := sourceLineOffsets(source, start.Line)
	if !ok {
		return SourceRange{Start: start, End: end}
	}
	offset := lineStart
	for range start.Column - 1 {
		if offset >= lineEnd {
			break
		}
		_, width := utf8.DecodeRune(source[offset:lineEnd])
		offset += width
	}
	for offset < lineEnd && (source[offset] == ' ' || source[offset] == '\t') {
		offset++
	}
	endOffset := offset
	if endOffset < lineEnd && isSourceIdentifierByte(source[endOffset]) {
		for endOffset < lineEnd && isSourceIdentifierByte(source[endOffset]) {
			endOffset++
		}
		for endOffset+1 < lineEnd && source[endOffset] == '.' && isSourceIdentifierByte(source[endOffset+1]) {
			endOffset++
			for endOffset < lineEnd && isSourceIdentifierByte(source[endOffset]) {
				endOffset++
			}
		}
	} else {
		for endOffset < lineEnd {
			value, width := utf8.DecodeRune(source[endOffset:lineEnd])
			if strings.ContainsRune(" \t,.:;(){}[]<>?=@", value) {
				break
			}
			endOffset += width
		}
	}
	if endOffset == offset && endOffset < lineEnd {
		_, width := utf8.DecodeRune(source[endOffset:lineEnd])
		endOffset += width
	}
	end.Column = 1 + utf8.RuneCount(source[lineStart:endOffset])
	if end.Column <= start.Column {
		end.Column = start.Column + 1
	}
	return SourceRange{Start: start, End: end}
}

func isSourceIdentifierByte(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}
