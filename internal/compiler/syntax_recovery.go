package compiler

import (
	"bytes"
	"errors"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/analyzer"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

// ParseSourceRecovering returns the declarations that can be recovered from a
// source together with independent syntax diagnostics in source order. The
// initial token pass isolates top-level declarations so recovery reparses only
// the declaration containing an error, never the complete source file.
func ParseSourceRecovering(path string, source []byte) (*grammar.SkelContent, Diagnostics) {
	parsed, err := parser.ParseSourcePartial(path, source)
	if err == nil {
		return parsed.Content, nil
	}

	segments := splitSourceSegments(path, source)
	content := new(grammar.SkelContent)
	diagnostics := Diagnostics{}
	for _, segment := range segments {
		remaining := analyzer.MaxDiagnosticsPerDomain - len(diagnostics)
		fragment, fragmentDiagnostics := parseSourceSegmentRecovering(path, source, segment, remaining)
		diagnostics = append(diagnostics, fragmentDiagnostics...)
		orderDiagnostics := mergeRecoveredContent(path, source, content, fragment)
		remaining = analyzer.MaxDiagnosticsPerDomain - len(diagnostics)
		if len(orderDiagnostics) > remaining {
			orderDiagnostics = orderDiagnostics[:remaining]
		}
		diagnostics = append(diagnostics, orderDiagnostics...)
	}
	return content, diagnostics
}

func parseSourceSegmentRecovering(path string, source []byte, segment _SourceSegment, limit int) (*grammar.SkelContent, Diagnostics) {
	working := append([]byte{}, source[segment.start:segment.end]...)
	diagnostics := Diagnostics{}
	seen := map[string]bool{}
	for len(diagnostics) < limit {
		parsed, err := parser.ParseSourceFragment(path, working, segment.line, segment.start)
		if err == nil {
			return parsed.Content, diagnostics
		}
		diagnostic := syntaxDiagnostic(path, source, err, parsed.FinalizeError)
		key := diagnostic.Position.String() + "\x00" + diagnostic.Message
		if seen[key] {
			return parsed.Content, diagnostics
		}
		seen[key] = true
		diagnostics = append(diagnostics, diagnostic)
		localPosition := diagnostic.Position
		localPosition.Line -= segment.line - 1
		if !recoverSyntaxLine(&working, localPosition, diagnostic.Code == DiagnosticCodeSyntaxEOF) {
			return parsed.Content, diagnostics
		}
	}
	parsed, _ := parser.ParseSourceFragment(path, working, segment.line, segment.start)
	return parsed.Content, diagnostics
}

func mergeRecoveredContent(path string, original []byte, target, source *grammar.SkelContent) Diagnostics {
	if source == nil {
		return nil
	}
	if target.Pos == (lexer.Position{}) {
		target.Pos = source.Pos
	}
	diagnostics := Diagnostics{}
	if source.Domain != nil && target.Domain == nil && len(target.Imports) == 0 && len(target.Entries) == 0 {
		target.Domain = source.Domain
	} else if source.Domain != nil {
		diagnostics = append(diagnostics, declarationOrderDiagnostic(
			path, original, source.Domain.Pos, "domain declaration must be the first declaration in a file",
		))
	}
	if len(source.Imports) > 0 && len(target.Entries) > 0 {
		diagnostics = append(diagnostics, declarationOrderDiagnostic(
			path, original, source.Imports[0].Pos, "import declaration must appear before type and endpoint declarations",
		))
	} else {
		target.Imports = append(target.Imports, source.Imports...)
	}
	target.Entries = append(target.Entries, source.Entries...)
	return diagnostics
}

func declarationOrderDiagnostic(path string, source []byte, position lexer.Position, message string) Diagnostic {
	start := workspacePosition(position)
	start.File = path
	return Diagnostic{
		Code: DiagnosticCodeSyntaxUnexpected, Severity: DiagnosticSeverityError,
		Position: start, Range: sourceRangeAt(start, source), Message: message,
	}
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
	var unexpectedToken *participle.UnexpectedTokenError
	var unexpectedEOF *grammar.UnexpectedEOFError
	if finalize {
		code = DiagnosticCodeSyntaxFinalize
	} else if errors.As(err, &unexpectedEOF) || errors.As(err, &unexpectedToken) && unexpectedToken.Unexpected.EOF() {
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

func recoverSyntaxLine(source *[]byte, position model.Position, unexpectedEOF bool) bool {
	if position.Line <= 0 {
		return false
	}
	if unexpectedEOF {
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
