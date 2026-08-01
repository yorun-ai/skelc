package lsp

import (
	"encoding/json"

	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/compiler"
	"go.yorun.ai/skelc/internal/lsp/source"
)

func diagnosticSuggestionData(suggestion *compiler.DiagnosticSuggestion) protocol.LSPAny {
	if suggestion == nil {
		return nil
	}
	content, err := json.Marshal(suggestion)
	if err != nil {
		return nil
	}
	return protocol.LSPAny(content)
}

func sourceRangeToProtocol(content string, sourceRange compiler.SourceRange) protocol.Range {
	buffer := source.New(content)
	start := buffer.IdentifierRange(sourceRange.Start.Line, sourceRange.Start.Column, "").Start
	end := buffer.IdentifierRange(sourceRange.End.Line, sourceRange.End.Column, "").Start
	if comparePosition(end, start) <= 0 {
		end = start
		end.Character++
	}
	return protocol.Range{Start: start, End: end}
}

func diagnosticSeverityToProtocol(severity compiler.DiagnosticSeverity) protocol.DiagnosticSeverity {
	if severity == compiler.DiagnosticSeverityWarning {
		return protocol.DiagnosticSeverityWarning
	}
	return protocol.DiagnosticSeverityError
}

func comparePosition(left, right protocol.Position) int {
	if left.Line != right.Line {
		return int(left.Line) - int(right.Line)
	}
	return int(left.Character) - int(right.Character)
}
