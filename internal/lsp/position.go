package lsp

import (
	"encoding/json"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/alecthomas/participle/v2/lexer"
	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/parser"
)

func occurrenceAt(document *_Document, position protocol.Position) (_Occurrence, bool) {
	for _, occurrence := range document.Occurrences {
		if containsPosition(occurrence.Range, position) {
			return occurrence, true
		}
	}
	return _Occurrence{}, false
}

func diagnosticSuggestionData(suggestion *parser.DiagnosticSuggestion) protocol.LSPAny {
	if suggestion == nil {
		return nil
	}
	content, err := json.Marshal(suggestion)
	if err != nil {
		return nil
	}
	return protocol.LSPAny(content)
}

func sourceRangeToProtocol(source string, sourceRange parser.SourceRange) protocol.Range {
	start := identifierRange(source, lexer.Position{
		Line: sourceRange.Start.Line, Column: sourceRange.Start.Column,
	}, "").Start
	end := identifierRange(source, lexer.Position{
		Line: sourceRange.End.Line, Column: sourceRange.End.Column,
	}, "").Start
	if comparePosition(end, start) <= 0 {
		end = start
		end.Character++
	}
	return protocol.Range{Start: start, End: end}
}

func diagnosticSeverityToProtocol(severity parser.DiagnosticSeverity) protocol.DiagnosticSeverity {
	if severity == parser.DiagnosticSeverityWarning {
		return protocol.DiagnosticSeverityWarning
	}
	return protocol.DiagnosticSeverityError
}

func containsPosition(r protocol.Range, position protocol.Position) bool {
	return comparePosition(r.Start, position) <= 0 && comparePosition(position, r.End) < 0
}

func comparePosition(left, right protocol.Position) int {
	if left.Line != right.Line {
		return int(left.Line) - int(right.Line)
	}
	return int(left.Character) - int(right.Character)
}

func identifierRange(source string, pos lexer.Position, name string) protocol.Range {
	line := max(pos.Line-1, 0)
	lineSource := sourceLine(source, line)
	byteColumn := 0
	for range max(pos.Column-1, 0) {
		if byteColumn >= len(lineSource) {
			break
		}
		_, width := utf8.DecodeRuneInString(lineSource[byteColumn:])
		byteColumn += width
	}
	start := protocol.Position{Line: uint32(line), Character: uint32(utf16Length(lineSource[:byteColumn]))}
	return protocol.Range{Start: start, End: protocol.Position{Line: start.Line, Character: start.Character + uint32(utf16Length(name))}}
}

func offsetRange(source string, start, end int) protocol.Range {
	return protocol.Range{Start: offsetPosition(source, start), End: offsetPosition(source, end)}
}

func offsetPosition(source string, offset int) protocol.Position {
	offset = min(max(offset, 0), len(source))
	line := strings.Count(source[:offset], "\n")
	lineStart := strings.LastIndexByte(source[:offset], '\n') + 1
	return protocol.Position{Line: uint32(line), Character: uint32(utf16Length(source[lineStart:offset]))}
}

func sourceLine(source string, line int) string {
	lines := strings.Split(source, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return strings.TrimSuffix(lines[line], "\r")
}

func utf16Length(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func firstRune(value string) rune {
	r, _ := utf8.DecodeRuneInString(value)
	return r
}

func sortLocations(locations []protocol.Location) {
	slices.SortFunc(locations, func(left, right protocol.Location) int {
		if compared := strings.Compare(string(left.URI), string(right.URI)); compared != 0 {
			return compared
		}
		return comparePosition(left.Range.Start, right.Range.Start)
	})
}
