package grammar

import (
	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var skelLexerRules = []lexer.SimpleRule{
	{Name: "Whitespace", Pattern: `[ \t]+`},
	{Name: "LineComment", Pattern: `//[^\n\r]*`},
	{Name: "BlockComment", Pattern: `(?s)/\*.*?\*/`},
	{Name: "Newline", Pattern: `[\n\r]+`},
	{Name: "TripleString", Pattern: `(?s)""".*?"""`},
	{Name: "String", Pattern: `(?s)"([^"\\]|\\.)*"`},
	{Name: "Number", Pattern: `[0-9]+(\.[0-9]+)?`},
	{Name: "Identifier", Pattern: `[_a-zA-Z][_a-zA-Z0-9]*`},
	{Name: "Punctuation", Pattern: `[@\(\)\{\}\<\>\[\].,*:;=?"]`},
}

var skelLexer = lexer.MustSimple(skelLexerRules)

// LexerDefinition returns the lossless lexer used by the Skel parser.
func LexerDefinition() lexer.Definition {
	return skelLexer
}

var Options = []participle.Option{
	participle.UseLookahead(8),
	participle.Lexer(skelLexer),
	participle.Elide("Whitespace"),
	participle.Elide("LineComment"),
	participle.Elide("BlockComment"),
}
