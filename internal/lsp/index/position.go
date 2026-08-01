package index

import (
	"github.com/alecthomas/participle/v2/lexer"
	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/lsp/source"
)

func identifierRange(content string, position lexer.Position, name string) protocol.Range {
	return source.New(content).IdentifierRange(position.Line, position.Column, name)
}
