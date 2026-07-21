package grammar

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Enum struct {
	Pos        lexer.Position
	Decorators []*Decorator
	Pub        bool
	Name       *Identifier `parser:"@@"`
	Items      []*EnumItem `parser:"\"{\" (Newline)* (@@ (Newline)*)* \"}\""`
}

type EnumItem struct {
	Pos        lexer.Position
	Decorators []*Decorator `parser:"(@@ (Newline)*)*"`
	Name       *Identifier  `parser:"@@"`
}

type Data struct {
	Pos            lexer.Position
	Decorators     []*Decorator
	Pub            bool
	Name           *Identifier      `parser:"@@"`
	Qualifier      *Identifier      `parser:"@@?"`
	TypeParameters []*TypeParameter `parser:"(\"<\" (@@ (\",\")?)* \">\")?"`
	Members        []*DataMember    `parser:"\"{\" (Newline)* (@@ (Newline)*)* \"}\""`
}

type TypeParameter struct {
	Pos      lexer.Position
	Name     *Identifier `parser:"@@"`
	Nullable bool        `parser:"@\"?\"?"`
}

type DataMember struct {
	Pos        lexer.Position
	Decorators []*Decorator `parser:"(@@ (Newline)*)*"`
	Name       *Identifier  `parser:"@@"`
	Type       *Type        `parser:"\":\" @@"`
}

type Event struct {
	Pos            lexer.Position
	Decorators     []*Decorator
	Pub            bool
	Qualifier      *Identifier
	TypeParameters []*TypeParameter
	Name           *Identifier   `parser:"@@"`
	Payload        *EventPayload `parser:"\"{\" (Newline)* @@ (Newline)* \"}\""`
}

type EventPayload struct {
	Pos     lexer.Position
	Members []*DataMember `parser:"\"payload\" \"{\" (Newline)* (@@ (Newline)*)* \"}\""`
}

type Service struct {
	Pos        lexer.Position
	Decorators []*Decorator
	Pub        bool
	Name       *Identifier       `parser:"@@"`
	Sections   []*ServiceSection `parser:"\"{\" (Newline)* (@@ (Newline)*)* \"}\""`

	Audiences []*ServiceAudience
	Auth      *AuthMarker
	Methods   []*Method
}

type ServiceSection struct {
	Decorators []*Decorator     `parser:"(@@ (Newline)*)*"`
	Audience   *ServiceAudience `parser:"  (?= \"for\") @@"`
	Auth       *AuthMarker      `parser:"| (?= (\"auth\" | \"noauth\")) @@"`
	Require    *Require         `parser:"| (?= \"require\") @@"`
	Method     *Method          `parser:"| @@"`
}

type ServiceAudience struct {
	Pos     lexer.Position
	Keyword string         `parser:"@\"for\""`
	Actor   *QualifiedName `parser:"@@"`
	Via     *Identifier    `parser:"(\"via\" @@)?"`
}

type Web struct {
	Pos        lexer.Position
	Decorators []*Decorator
	Name       *Identifier    `parser:"@@"`
	Audiences  []*WebAudience `parser:"\"{\" (Newline)* (@@ (Newline)*)* \"}\""`
}

type WebAudience struct {
	Pos     lexer.Position
	Keyword string         `parser:"@\"for\""`
	Actor   *QualifiedName `parser:"@@"`
	Via     *Identifier    `parser:"(\"via\" @@)?"`
}

type Actor struct {
	Pos        lexer.Position
	Decorators []*Decorator
	Pub        bool
	Name       *Identifier     `parser:"@@"`
	Vias       []*ActorVia     `parser:"\"{\" (Newline)* (@@ (Newline)*)*"`
	Sections   []*ActorSection `parser:"(@@ (Newline)*)* \"}\""`
}

func (actor *Actor) Finalize() error {
	for _, section := range actor.Sections {
		if section.Auth == nil {
			continue
		}
		if err := section.Auth.Finalize(); err != nil {
			return err
		}
	}
	return nil
}

type ActorVia struct {
	Pos  lexer.Position
	Name *Identifier `parser:"\"via\" @@ \"{\" (Newline)* \"}\""`
}

type ActorSection struct {
	Auth       *ActorAuth       `parser:"  (?= \"auth\") @@"`
	Permission *ActorPermission `parser:"| (?= \"permission\") @@"`
}

type ActorAuth struct {
	Pos      lexer.Position
	Sections []*ActorAuthSection `parser:"\"auth\" \"{\" (Newline)* (@@ (Newline)*)* \"}\""`

	Credential *ActorCredential
	Info       *ActorInfo
}

func (auth *ActorAuth) Finalize() error {
	for _, section := range auth.Sections {
		switch {
		case section.Credential != nil:
			if auth.Credential != nil {
				return participle.Errorf(section.Credential.Pos, "duplicated actor auth credential")
			}
			auth.Credential = section.Credential
		case section.Info != nil:
			if auth.Info != nil {
				return participle.Errorf(section.Info.Pos, "duplicated actor auth info")
			}
			auth.Info = section.Info
		}
	}
	return nil
}

type ActorAuthSection struct {
	Credential *ActorCredential `parser:"  (?= \"credential\") @@"`
	Info       *ActorInfo       `parser:"| (?= \"info\") @@"`
}

type ActorCredential struct {
	Pos     lexer.Position
	Members []*DataMember `parser:"\"credential\" \"{\" (Newline)* (@@ (Newline)*)* \"}\""`
}

type ActorInfo struct {
	Pos     lexer.Position
	Members []*DataMember `parser:"\"info\" \"{\" (Newline)* (@@ (Newline)*)* \"}\""`
}

type ActorPermission struct {
	Pos lexer.Position
	End string `parser:"\"permission\" \"{\" (Newline)* @\"}\""`
}

type Resource struct {
	Pos        lexer.Position
	Decorators []*Decorator
	Name       *Identifier        `parser:"@@"`
	Sections   []*ResourceSection `parser:"\"{\" (Newline)* (@@ (Newline)*)* \"}\""`

	Checks  []*ResourceCheck
	Actions []*ResourceAction
}

func (resource *Resource) Finalize() {
	for _, section := range resource.Sections {
		if section.Check != nil {
			resource.Checks = append(resource.Checks, section.Check)
		}
		if section.Action != nil {
			resource.Actions = append(resource.Actions, section.Action)
		}
	}
}

type ResourceSection struct {
	Check  *ResourceCheck  `parser:"  (?= \"check\") @@"`
	Action *ResourceAction `parser:"| @@"`
}

type ResourceAction struct {
	Pos        lexer.Position
	Decorators []*Decorator     `parser:"(@@ (Newline)*)*"`
	Name       *Identifier      `parser:"\"action\" @@"`
	Checks     []*ResourceCheck `parser:"(\"{\" (Newline)* (@@ (Newline)*)* \"}\")?"`
}

type ResourceCheck struct {
	Pos       lexer.Position
	Name      *Identifier `parser:"\"check\" @@"`
	Arguments []*Argument `parser:"\"(\" (@@ (\",\")?)* \")\""`
}

type Task struct {
	Pos        lexer.Position
	Decorators []*Decorator
	Name       *Identifier    `parser:"@@"`
	Triggers   []*TaskTrigger `parser:"\"{\" (Newline)* (@@ (Newline)*)* \"}\""`
}

type Decorator struct {
	Pos   lexer.Position
	Name  *Identifier     `parser:"\"@\" @@"`
	Value *DecoratorValue `parser:"(\"(\" @@ \")\")?"`
}

type DecoratorValue struct {
	Raw string
}

func (v *DecoratorValue) Parse(lex *lexer.PeekingLexer) error {
	var raw strings.Builder
	depth := 0

	for {
		token := lex.Peek()
		if token.EOF() {
			return participle.Errorf(token.Pos, "unexpected EOF in decorator value")
		}
		if token.Value == ")" && depth == 0 {
			break
		}

		lex.Next()
		switch token.Value {
		case "(":
			depth++
		case ")":
			depth--
		}
		raw.WriteString(token.Value)
	}

	v.Raw = strings.TrimSpace(raw.String())
	return nil
}
