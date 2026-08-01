package source

import "go.lsp.dev/protocol"

// ComparePosition compares two LSP positions in source order.
func ComparePosition(left, right protocol.Position) int {
	if left.Line != right.Line {
		return int(left.Line) - int(right.Line)
	}
	return int(left.Character) - int(right.Character)
}

// ContainsPosition reports whether position lies in the half-open range.
func ContainsPosition(sourceRange protocol.Range, position protocol.Position) bool {
	return ComparePosition(sourceRange.Start, position) <= 0 && ComparePosition(position, sourceRange.End) < 0
}
