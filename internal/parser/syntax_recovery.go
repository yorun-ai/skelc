package parser

import (
	"bytes"
	"errors"
	"strings"

	"github.com/alecthomas/participle/v2"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

// ParseSourceRecovering returns the declarations that can be recovered from a
// source together with independent syntax diagnostics in source order.
func ParseSourceRecovering(path string, source []byte) (*grammar.SkelContent, Diagnostics) {
	working := append([]byte{}, source...)
	diagnostics := Diagnostics{}
	seen := map[string]bool{}
	for len(diagnostics) < analyzer.MaxDiagnosticsPerDomain {
		content, err, finalize := parseSourceOnce(path, working)
		if err == nil {
			return content, diagnostics
		}
		diagnostic := syntaxDiagnostic(path, source, err, finalize)
		key := diagnostic.Position.String() + "\x00" + diagnostic.Message
		if seen[key] {
			return content, diagnostics
		}
		seen[key] = true
		diagnostics = append(diagnostics, diagnostic)
		if !recoverSyntaxLine(&working, diagnostic.Position, diagnostic.Message) {
			return content, diagnostics
		}
	}
	return nil, diagnostics
}

func parseSourceOnce(path string, source []byte) (*grammar.SkelContent, error, bool) {
	content, err := sourceParser.Parse(path, bytes.NewReader(source))
	if err != nil {
		return content, err, false
	}
	if err := content.Finalize(); err != nil {
		return content, err, true
	}
	if content.Domain != nil {
		if err := content.Domain.Finalize(); err != nil {
			return content, err, true
		}
	}
	return content, nil, false
}

func syntaxDiagnostic(path string, source []byte, err error, finalize bool) Diagnostic {
	position := model.Position{File: path, Line: 1, Column: 1}
	message := err.Error()
	code := DiagnosticCodeSyntaxUnexpected
	var parseError participle.Error
	if errors.As(err, &parseError) {
		position = workspacePosition(parseError.Position())
		message = parseError.Message()
	}
	if finalize {
		code = DiagnosticCodeSyntaxFinalize
	} else if strings.Contains(strings.ToLower(message), "eof") {
		code = DiagnosticCodeSyntaxEOF
	}
	diagnostic := Diagnostic{
		Code: code, Severity: DiagnosticSeverityError, Position: position,
		Range: sourceRangeAt(position, source), Message: message,
	}
	if expected := expectedSyntaxReplacement(message); expected != "" {
		diagnostic.Suggestion = &DiagnosticSuggestion{Message: "insert " + expected, Replacement: expected}
	}
	if lineStart, lineEnd, ok := sourceLineOffsets(source, position.Line); ok &&
		braceBalance(source[:lineStart]) > 0 && looksLikeTopLevelDeclaration(strings.TrimSpace(string(source[lineStart:lineEnd]))) {
		rangePosition := model.Position{File: path, Line: position.Line, Column: 1}
		diagnostic.Range = sourceRangeAt(rangePosition, source)
		diagnostic.Suggestion = &DiagnosticSuggestion{Message: "insert } before this declaration", Replacement: "}\n"}
	}
	return diagnostic
}

func expectedSyntaxReplacement(message string) string {
	marker := "expected \""
	index := strings.LastIndex(message, marker)
	if index < 0 {
		return ""
	}
	remainder := message[index+len(marker):]
	end := strings.IndexByte(remainder, '"')
	if end < 0 {
		return ""
	}
	value := remainder[:end]
	if strings.ContainsAny(value, " \t\r\n") {
		return ""
	}
	return value
}

func recoverSyntaxLine(source *[]byte, position model.Position, message string) bool {
	if position.Line <= 0 {
		return false
	}
	if strings.Contains(strings.ToLower(message), "eof") {
		if start, end, found := unclosedDecoratorOffsets(*source); found {
			blankBytes((*source)[start:end])
			return true
		}
		balance := braceBalance(*source)
		if balance > 0 {
			*source = append(*source, []byte("\n"+strings.Repeat("}", balance))...)
			return true
		}
	}
	lineStart, lineEnd, ok := sourceLineOffsets(*source, position.Line)
	if !ok {
		return false
	}
	line := strings.TrimSpace(string((*source)[lineStart:lineEnd]))
	if braceBalance((*source)[:lineStart]) > 0 && looksLikeTopLevelDeclaration(line) {
		previousStart, previousEnd, previous := sourceLineOffsets(*source, position.Line-1)
		if previous {
			blankBytes((*source)[previousStart:previousEnd])
			if previousStart < previousEnd {
				(*source)[previousStart] = '}'
				return true
			}
		}
	}
	if lineStart == lineEnd {
		return false
	}
	blankBytes((*source)[lineStart:lineEnd])
	return true
}

func unclosedDecoratorOffsets(source []byte) (int, int, bool) {
	lines := bytes.Split(source, []byte{'\n'})
	offset := 0
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] == '@' && bytes.Count(trimmed, []byte{'('}) > bytes.Count(trimmed, []byte{')'}) {
			return offset, offset + len(line), true
		}
		offset += len(line) + 1
	}
	return 0, 0, false
}

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

func blankBytes(value []byte) {
	for index := range value {
		if value[index] != '\r' && value[index] != '\n' {
			value[index] = ' '
		}
	}
}

func braceBalance(source []byte) int {
	balance := 0
	for _, value := range source {
		switch value {
		case '{':
			balance++
		case '}':
			if balance > 0 {
				balance--
			}
		}
	}
	return balance
}

func looksLikeTopLevelDeclaration(line string) bool {
	line = strings.TrimPrefix(line, "pub ")
	for _, prefix := range []string{"domain ", "import ", "enum ", "data ", "config ", "actor ", "resource ", "service ", "web ", "event ", "task ", "@"} {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
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
	offset := min(lineStart+start.Column-1, lineEnd)
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
		for endOffset < lineEnd && !strings.ContainsRune(" \t,.:;(){}[]<>?=@", rune(source[endOffset])) {
			endOffset++
		}
	}
	if endOffset == offset && endOffset < lineEnd {
		endOffset++
	}
	end.Column = start.Column + max(endOffset-offset, 1)
	return SourceRange{Start: start, End: end}
}

func isSourceIdentifierByte(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}
