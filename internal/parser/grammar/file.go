package grammar

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type SkelContent struct {
	Pos     lexer.Position
	Domain  *DomainContent `parser:"(Newline* @@)?"`
	Imports []*ImportDecl  `parser:"(Newline* @@)*"`
	Entries []*SkelEntry   `parser:"(Newline* @@)* Newline*"`
}

func (content *SkelContent) Finalize() error {
	if content == nil {
		return nil
	}
	for _, entry := range content.Entries {
		switch {
		case entry.Enum != nil:
			entry.Enum.Decorators = entry.Decorators
			entry.Enum.Pub = entry.Pub
		case entry.Data != nil:
			entry.Data.Decorators = entry.Decorators
			entry.Data.Pub = entry.Pub
		case entry.Config != nil:
			entry.Config.Decorators = entry.Decorators
			entry.Config.Pub = entry.Pub
		case entry.Actor != nil:
			entry.Actor.Decorators = entry.Decorators
			entry.Actor.Pub = entry.Pub
			if err := entry.Actor.Finalize(); err != nil {
				return err
			}
		case entry.Resource != nil:
			entry.Resource.Decorators = entry.Decorators
			entry.Resource.Finalize()
		case entry.Service != nil:
			entry.Service.Decorators = entry.Decorators
			entry.Service.Pub = entry.Pub
		case entry.Web != nil:
			entry.Web.Decorators = entry.Decorators
		case entry.Event != nil:
			entry.Event.Decorators = entry.Decorators
			entry.Event.Pub = entry.Pub
		case entry.Task != nil:
			entry.Task.Decorators = entry.Decorators
		}
	}
	return nil
}

type DomainContent struct {
	Pos         lexer.Position
	Decorators  []*Decorator   `parser:"(@@ (Newline)*)*"`
	Name        *QualifiedName `parser:"\"domain\" @@"`
	Description string
}

type ImportDecl struct {
	Pos    lexer.Position
	Domain *QualifiedName `parser:"\"import\" @@"`
	Alias  *Identifier    `parser:"(\"as\" @@)?"`
}

type QualifiedName struct {
	Pos   lexer.Position
	Parts []*Identifier `parser:"@@ (\".\" @@)*"`
}

func (name *QualifiedName) String() string {
	if name == nil {
		return ""
	}
	partValues := make([]string, 0, len(name.Parts))
	for _, part := range name.Parts {
		partValues = append(partValues, part.Value)
	}
	return strings.Join(partValues, ".")
}

func (content *DomainContent) Finalize() error {
	if content == nil {
		return nil
	}
	for _, decorator := range content.Decorators {
		switch decorator.Name.Value {
		case "desc":
			if content.Description != "" {
				return participle.Errorf(decorator.Name.Pos, "duplicated decorator @desc")
			}
			if decorator.Value == nil {
				return participle.Errorf(decorator.Name.Pos, "decorator @desc requires a string argument")
			}
			description, err := UnquoteDescriptionString(decorator.Value.Raw)
			if err != nil {
				return participle.Errorf(decorator.Name.Pos, "%s", err)
			}
			content.Description = description
		default:
			return participle.Errorf(decorator.Name.Pos, "unexpected decorator @%s", decorator.Name.Value)
		}
	}
	return nil
}

type SkelEntry struct {
	Pos        lexer.Position
	Decorators []*Decorator `parser:"(@@ (Newline)*)*"`
	Pub        bool         `parser:"@\"pub\"?"`
	Enum       *Enum        `parser:"(\"enum\" @@"`
	Data       *Data        `parser:"| \"data\" @@"`
	Config     *Data        `parser:"| \"config\" @@"`
	Actor      *Actor       `parser:"| \"actor\" @@"`
	Resource   *Resource    `parser:"| \"resource\" @@"`
	Service    *Service     `parser:"| \"service\" @@"`
	Web        *Web         `parser:"| \"web\" @@"`
	Event      *Event       `parser:"| \"event\" @@"`
	Task       *Task        `parser:"| \"task\" @@)"`
}
