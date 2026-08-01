package index

import (
	"github.com/alecthomas/participle/v2/lexer"
	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/lsp/source"
)

func identifierRange(content string, position lexer.Position, name string) protocol.Range {
	return source.New(content).IdentifierRange(position.Line, position.Column, name)
}

func comparePosition(left, right protocol.Position) int {
	if left.Line != right.Line {
		return int(left.Line) - int(right.Line)
	}
	return int(left.Character) - int(right.Character)
}
