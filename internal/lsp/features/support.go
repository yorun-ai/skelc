package features

import (
	"slices"
	"strings"

	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/lsp/index"
	"go.yorun.ai/skelc/internal/lsp/source"
)

func occurrenceAt(document *index.Document, position protocol.Position) (index.Occurrence, bool) {
	for _, occurrence := range document.Occurrences {
		if containsPosition(occurrence.Range, position) {
			return occurrence, true
		}
	}
	return index.Occurrence{}, false
}

func positionOffset(content string, position protocol.Position) int {
	return source.New(content).Offset(position)
}

func positionInNonCode(content string, position protocol.Position) bool {
	return source.New(content).InNonCode(position)
}

func offsetPosition(content string, offset int) protocol.Position {
	return source.New(content).Position(offset)
}

func offsetRange(content string, start, end int) protocol.Range {
	return source.New(content).Range(start, end)
}

func utf16Length(value string) int {
	return source.UTF16Length(value)
}

func scanIdentifiers(content string) []source.Token {
	return source.New(content).IdentifierTokens()
}

func isIdentifierValue(value string) bool {
	return index.IsIdentifier(value)
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

func sortLocations(locations []protocol.Location) {
	slices.SortFunc(locations, func(left, right protocol.Location) int {
		if compared := strings.Compare(string(left.URI), string(right.URI)); compared != 0 {
			return compared
		}
		return comparePosition(left.Range.Start, right.Range.Start)
	})
}
