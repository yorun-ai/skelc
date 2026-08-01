package parser

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

var sourceParser = participle.MustBuild[grammar.SkelContent](grammar.Options...)

// SourceParseResult preserves partially parsed content and identifies whether
// a failure came from syntax-tree finalization rather than token parsing.
type SourceParseResult struct {
	// Content is the complete or partially parsed syntax tree.
	Content *grammar.SkelContent
	// FinalizeError reports whether the returned error came from syntax-tree finalization.
	FinalizeError bool
}

// ParseSource parses and finalizes one Skel source file without resolving its
// imports or performing domain-level semantic analysis.
func ParseSource(path string, source []byte) (*grammar.SkelContent, error) {
	result, err := ParseSourcePartial(path, source)
	return result.Content, err
}

// ParseSourcePartial parses one source and preserves partial content and error
// phase information for compiler recovery.
func ParseSourcePartial(path string, source []byte) (SourceParseResult, error) {
	content, err := sourceParser.Parse(path, bytes.NewReader(source))
	return finalizeSource(content, err)
}

// ParseSourceFragment parses a source fragment while retaining its original
// line and byte offsets for compiler recovery.
func ParseSourceFragment(path string, source []byte, line, offset int) (SourceParseResult, error) {
	lex, err := grammar.LexerDefinition().Lex(path, bytes.NewReader(source))
	if err != nil {
		return SourceParseResult{}, err
	}
	adjusted := &_OffsetLexer{lexer: lex, lineOffset: line - 1, byteOffset: offset}
	symbols := grammar.LexerDefinition().Symbols()
	peeking, err := lexer.Upgrade(adjusted, symbols["Whitespace"], symbols["LineComment"], symbols["BlockComment"])
	if err != nil {
		return SourceParseResult{}, err
	}
	content, err := sourceParser.ParseFromLexer(peeking)
	return finalizeSource(content, err)
}

// ValidateSource validates the grammar and finalized syntax state of one Skel source file.
func ValidateSource(path string, source []byte) error {
	_, err := ParseSource(path, source)
	if err != nil {
		return fmt.Errorf("parse %s failed: %w", path, err)
	}
	return nil
}

func finalizeSource(content *grammar.SkelContent, parseErr error) (SourceParseResult, error) {
	result := SourceParseResult{Content: content}
	if parseErr != nil {
		return result, parseErr
	}
	if err := content.Finalize(); err != nil {
		result.FinalizeError = true
		return result, err
	}
	if content.Domain != nil {
		if err := content.Domain.Finalize(); err != nil {
			result.FinalizeError = true
			return result, err
		}
	}
	return result, nil
}

type _OffsetLexer struct {
	lexer      lexer.Lexer
	lineOffset int
	byteOffset int
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
