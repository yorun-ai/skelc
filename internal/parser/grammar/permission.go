package grammar

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Method struct {
	Pos        lexer.Position
	Decorators []*Decorator  `parser:"(@@ (Newline)*)*"`
	Name       *Identifier   `parser:"\"method\" @@"`
	Auth       *AuthMarker   `parser:"\"{\" (Newline)* (@@ (Newline)*)?"`
	Require    *Require      `parser:"(@@ (Newline)*)?"`
	Input      *MethodInput  `parser:"(@@ (Newline)*)?"`
	Output     *MethodOutput `parser:"(@@ (Newline)*)? \"}\""`
}

type Require struct {
	Pos  lexer.Position
	Expr *RequireExpr `parser:"\"require\" @@"`
}

type RequireExpr struct {
	Mode     string
	Children []*RequireExpr
	Term     *RequireTerm
}

func (expr *RequireExpr) Parse(lex *lexer.PeekingLexer) error {
	skipNewlines(lex)
	first, err := parseIdentifier(lex)
	if err != nil {
		return err
	}

	if (first.Value == "all" || first.Value == "any") && lex.Peek().Value == "(" {
		expr.Mode = first.Value
		children, err := parseRequireExprChildren(lex)
		if err != nil {
			return err
		}
		expr.Children = children
		return nil
	}

	term, err := parseRequireTermFromFirst(lex, first)
	if err != nil {
		return err
	}
	expr.Term = term
	return nil
}

type RequireTerm struct {
	Target *PermissionTarget
	Call   *PermissionCall
}

func (term *RequireTerm) Parse(lex *lexer.PeekingLexer) error {
	skipNewlines(lex)
	first, err := parseIdentifier(lex)
	if err != nil {
		return err
	}
	parsed, err := parseRequireTermFromFirst(lex, first)
	if err != nil {
		return err
	}
	term.Target = parsed.Target
	term.Call = parsed.Call
	return nil
}

func parseRequireTermFromFirst(lex *lexer.PeekingLexer, first *Identifier) (*RequireTerm, error) {
	term := &RequireTerm{}
	target, checkName, err := parsePermissionTargetFromFirst(lex, first, true)
	if err != nil {
		return nil, err
	}
	if checkName != nil {
		if lex.Peek().Value != "(" {
			return nil, participle.Errorf(checkName.Pos, "permission check must be called")
		}
		callName := checkName
		call, err := parsePermissionCall(lex, callName)
		if err != nil {
			return nil, err
		}
		term.Call = call
	} else if lex.Peek().Value == "(" {
		return nil, participle.Errorf(first.Pos, "permission check must be resource:action:check")
	}
	term.Target = target
	return term, nil
}

type PermissionCode struct {
	Target *PermissionTarget `parser:"@@"`
}

type PermissionTarget struct {
	Resource *QualifiedName
	Action   *Identifier
}

func (target *PermissionTarget) Parse(lex *lexer.PeekingLexer) error {
	first, err := parseIdentifier(lex)
	if err != nil {
		return err
	}
	parsed, checkName, err := parsePermissionTargetFromFirst(lex, first, false)
	if err != nil {
		return err
	}
	if checkName != nil {
		return participle.Errorf(checkName.Pos, "permission code must be resource:action")
	}
	target.Resource = parsed.Resource
	target.Action = parsed.Action
	return nil
}

type PermissionCall struct {
	Name      *Identifier      `parser:"@@"`
	Arguments []*PermissionArg `parser:"\"(\" (@@ (\",\")?)* \")\""`
}

type PermissionArg struct {
	Pos   lexer.Position
	Parts []*Identifier `parser:"@@ (\".\" @@)*"`
	Raw   string
}

func (arg *PermissionArg) String() string {
	if arg == nil {
		return ""
	}
	if arg.Raw != "" {
		return arg.Raw
	}
	parts := make([]string, 0, len(arg.Parts))
	for _, part := range arg.Parts {
		parts = append(parts, part.Value)
	}
	return strings.Join(parts, ".")
}

func parseQualifiedIdentifierParts(lex *lexer.PeekingLexer) ([]*Identifier, error) {
	first, err := parseIdentifier(lex)
	if err != nil {
		return nil, err
	}
	parts := []*Identifier{first}
	for lex.Peek().Value == "." {
		lex.Next()
		part, err := parseIdentifier(lex)
		if err != nil {
			return nil, err
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func parsePermissionTargetFromFirst(lex *lexer.PeekingLexer, first *Identifier, allowCheck bool) (*PermissionTarget, *Identifier, error) {
	resourceParts := []*Identifier{first}
	for lex.Peek().Value == "." {
		lex.Next()
		part, err := parseIdentifier(lex)
		if err != nil {
			return nil, nil, err
		}
		resourceParts = append(resourceParts, part)
	}
	if token := lex.Next(); token.Value != ":" {
		return nil, nil, participle.Errorf(token.Pos, `expected ":"`)
	}
	action, err := parseIdentifier(lex)
	if err != nil {
		return nil, nil, err
	}
	var check *Identifier
	if lex.Peek().Value == ":" {
		if !allowCheck {
			return nil, nil, participle.Errorf(lex.Peek().Pos, "permission code must be resource:action")
		}
		lex.Next()
		check, err = parseIdentifier(lex)
		if err != nil {
			return nil, nil, err
		}
	}
	return &PermissionTarget{
		Resource: &QualifiedName{
			Pos:   first.Pos,
			Parts: resourceParts,
		},
		Action: action,
	}, check, nil
}

func parsePermissionCall(lex *lexer.PeekingLexer, name *Identifier) (*PermissionCall, error) {
	if token := lex.Next(); token.Value != "(" {
		return nil, participle.Errorf(token.Pos, `expected "("`)
	}

	call := &PermissionCall{Name: name}
	skipNewlines(lex)
	for lex.Peek().Value != ")" {
		arg, err := parsePermissionArg(lex)
		if err != nil {
			return nil, err
		}
		call.Arguments = append(call.Arguments, arg)
		skipNewlines(lex)
		if lex.Peek().Value == "," {
			lex.Next()
			skipNewlines(lex)
		}
	}

	lex.Next()
	return call, nil
}

func parsePermissionArg(lex *lexer.PeekingLexer) (*PermissionArg, error) {
	first, err := parseIdentifier(lex)
	if err != nil {
		return nil, err
	}
	parts := []*Identifier{first}
	values := []string{}
	for {
		part := parts[len(parts)-1]
		value := part.Value
		if lex.Peek().Value == "[" {
			lex.Next()
			if token := lex.Next(); token.Value != "*" {
				return nil, participle.Errorf(token.Pos, `expected "*"`)
			}
			if token := lex.Next(); token.Value != "]" {
				return nil, participle.Errorf(token.Pos, `expected "]"`)
			}
			value += "[*]"
		}
		values = append(values, value)
		if lex.Peek().Value != "." {
			break
		}
		lex.Next()
		next, err := parseIdentifier(lex)
		if err != nil {
			return nil, err
		}
		parts = append(parts, next)
	}
	return &PermissionArg{
		Pos:   parts[0].Pos,
		Parts: parts,
		Raw:   strings.Join(values, "."),
	}, nil
}

func parseRequireExprChildren(lex *lexer.PeekingLexer) ([]*RequireExpr, error) {
	if token := lex.Next(); token.Value != "(" {
		return nil, participle.Errorf(token.Pos, `expected "("`)
	}

	children := []*RequireExpr{}
	skipNewlines(lex)
	for lex.Peek().Value != ")" {
		child := &RequireExpr{}
		if err := child.Parse(lex); err != nil {
			return nil, err
		}
		children = append(children, child)
		skipNewlines(lex)
		if lex.Peek().Value == "," {
			lex.Next()
			skipNewlines(lex)
		}
	}
	lex.Next()
	return children, nil
}

func skipNewlines(lex *lexer.PeekingLexer) {
	for strings.Contains(lex.Peek().Value, "\n") || strings.Contains(lex.Peek().Value, "\r") {
		lex.Next()
	}
}
