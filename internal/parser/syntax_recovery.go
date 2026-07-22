package parser

import (
	"bytes"
	"errors"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/analyzer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

type _SourceSegment struct {
	start int
	end   int
	line  int
}

type _OffsetLexer struct {
	lexer      lexer.Lexer
	lineOffset int
	byteOffset int
}

// ParseSourceRecovering returns the declarations that can be recovered from a
// source together with independent syntax diagnostics in source order. The
// initial token pass isolates top-level declarations so recovery reparses only
// the declaration containing an error, never the complete source file.
func ParseSourceRecovering(path string, source []byte) (*grammar.SkelContent, Diagnostics) {
	segments := splitSourceSegments(path, source)
	content := new(grammar.SkelContent)
	diagnostics := Diagnostics{}
	for _, segment := range segments {
		remaining := analyzer.MaxDiagnosticsPerDomain - len(diagnostics)
		if remaining == 0 {
			break
		}
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
		content, err, finalize := parseSourceFragment(path, working, segment.line, segment.start)
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
		localPosition := diagnostic.Position
		localPosition.Line -= segment.line - 1
		if !recoverSyntaxLine(&working, localPosition, diagnostic.Message) {
			return content, diagnostics
		}
	}
	content, _, _ := parseSourceFragment(path, working, segment.line, segment.start)
	return content, diagnostics
}

func splitSourceSegments(path string, source []byte) []_SourceSegment {
	lex, err := grammar.LexerDefinition().Lex(path, bytes.NewReader(source))
	if err != nil {
		return []_SourceSegment{{end: len(source), line: 1}}
	}
	tokens, err := lexer.ConsumeAll(lex)
	if err != nil {
		return []_SourceSegment{{end: len(source), line: 1}}
	}

	symbols := grammar.LexerDefinition().Symbols()
	elided := map[lexer.TokenType]bool{
		symbols["Whitespace"]:   true,
		symbols["LineComment"]:  true,
		symbols["BlockComment"]: true,
		symbols["Newline"]:      true,
	}
	starts := []_SourceSegment{{line: 1}}
	depth := 0
	hasDeclaration := false
	lastLine := 0
	pendingDecoratorStart := -1
	pendingDecoratorLine := 0
	for index, token := range tokens {
		if token.EOF() || elided[token.Type] {
			continue
		}
		firstOnLine := token.Pos.Line != lastLine
		lastLine = token.Pos.Line
		if firstOnLine {
			kind := topLevelTokenKind(tokens, index, symbols["Identifier"], elided)
			if depth > 0 && kind == "decorator" {
				if pendingDecoratorStart < 0 {
					pendingDecoratorStart, _, _ = sourceLineOffsets(source, token.Pos.Line)
					pendingDecoratorLine = token.Pos.Line
				}
			} else if kind != "" {
				if hasDeclaration || depth > 0 {
					start, _, _ := sourceLineOffsets(source, token.Pos.Line)
					line := token.Pos.Line
					if depth > 0 && pendingDecoratorStart >= 0 {
						start = pendingDecoratorStart
						line = pendingDecoratorLine
					}
					starts = append(starts, _SourceSegment{start: start, line: line})
				}
				hasDeclaration = kind != "decorator"
				if depth > 0 {
					depth = 0
				}
				pendingDecoratorStart = -1
			} else {
				pendingDecoratorStart = -1
			}
		}
		switch token.Value {
		case "{":
			depth++
		case "}":
			if depth > 0 {
				depth--
			}
		}
	}
	for index := range starts {
		if index+1 < len(starts) {
			starts[index].end = starts[index+1].start
		} else {
			starts[index].end = len(source)
		}
	}
	return starts
}

func topLevelTokenKind(tokens []lexer.Token, index int, identifier lexer.TokenType, elided map[lexer.TokenType]bool) string {
	value := tokens[index].Value
	if value == "@" {
		return "decorator"
	}
	line := tokens[index].Pos.Line
	if value == "pub" {
		index = nextSignificantToken(tokens, index+1, line, elided)
		if index < 0 {
			return ""
		}
		value = tokens[index].Value
	}
	for _, keyword := range []string{"domain", "import", "enum", "data", "config", "actor", "resource", "service", "web", "event", "task"} {
		next := nextSignificantToken(tokens, index+1, line, elided)
		if value == keyword && next >= 0 && tokens[next].Type == identifier {
			return keyword
		}
	}
	return ""
}

func nextSignificantToken(tokens []lexer.Token, index, line int, elided map[lexer.TokenType]bool) int {
	for ; index < len(tokens); index++ {
		if tokens[index].EOF() || tokens[index].Pos.Line != line {
			return -1
		}
		if !elided[tokens[index].Type] {
			return index
		}
	}
	return -1
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

func parseSourceFragment(path string, source []byte, line, offset int) (*grammar.SkelContent, error, bool) {
	lex, err := grammar.LexerDefinition().Lex(path, bytes.NewReader(source))
	if err != nil {
		return nil, err, false
	}
	adjusted := &_OffsetLexer{lexer: lex, lineOffset: line - 1, byteOffset: offset}
	symbols := grammar.LexerDefinition().Symbols()
	peeking, err := lexer.Upgrade(adjusted, symbols["Whitespace"], symbols["LineComment"], symbols["BlockComment"])
	if err != nil {
		return nil, err, false
	}
	content, err := sourceParser.ParseFromLexer(peeking)
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

func (lex *_OffsetLexer) Next() (lexer.Token, error) {
	token, err := lex.lexer.Next()
	token.Pos.Line += lex.lineOffset
	token.Pos.Offset += lex.byteOffset
	var parseError participle.Error
	if errors.As(err, &parseError) {
		position := parseError.Position()
		position.Line += lex.lineOffset
		position.Offset += lex.byteOffset
		err = participle.Errorf(position, "%s", parseError.Message())
	}
	return token, err
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
