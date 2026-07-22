package lsp

import (
	"github.com/alecthomas/participle/v2/lexer"
	"go.lsp.dev/protocol"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

func entryDefinition(entry *grammar.SkelEntry) (string, lexer.Position, protocol.SymbolKind, string) {
	switch {
	case entry.Enum != nil:
		return entry.Enum.Name.Value, entry.Enum.Name.Pos, protocol.SymbolKindEnum, "enum"
	case entry.Data != nil:
		return entry.Data.Name.Value, entry.Data.Name.Pos, protocol.SymbolKindStruct, "data"
	case entry.Config != nil:
		return entry.Config.Name.Value, entry.Config.Name.Pos, protocol.SymbolKindStruct, "config"
	case entry.Actor != nil:
		return entry.Actor.Name.Value, entry.Actor.Name.Pos, protocol.SymbolKindInterface, "actor"
	case entry.Resource != nil:
		return entry.Resource.Name.Value, entry.Resource.Name.Pos, protocol.SymbolKindObject, "resource"
	case entry.Service != nil:
		return entry.Service.Name.Value, entry.Service.Name.Pos, protocol.SymbolKindInterface, "service"
	case entry.Web != nil:
		return entry.Web.Name.Value, entry.Web.Name.Pos, protocol.SymbolKindInterface, "web"
	case entry.Event != nil:
		return entry.Event.Name.Value, entry.Event.Name.Pos, protocol.SymbolKindEvent, "event"
	case entry.Task != nil:
		return entry.Task.Name.Value, entry.Task.Name.Pos, protocol.SymbolKindFunction, "task"
	default:
		return "", lexer.Position{}, protocol.SymbolKindNull, ""
	}
}

func entrySymbol(source string, entry *grammar.SkelEntry, name, detail, description string, kind protocol.SymbolKind, range_ protocol.Range) _Symbol {
	children := []_Symbol{}
	switch {
	case entry.Enum != nil:
		for _, item := range entry.Enum.Items {
			children = append(children, newSymbol(source, item.Name, "enum item", descriptionFromDecorators(item.Decorators), protocol.SymbolKindEnumMember, nil))
		}
	case entry.Data != nil:
		children = dataMemberSymbols(source, entry.Data.Members)
	case entry.Config != nil:
		children = dataMemberSymbols(source, entry.Config.Members)
	case entry.Actor != nil:
		for _, via := range entry.Actor.Vias {
			children = append(children, newSymbol(source, via.Name, "actor transport", "", protocol.SymbolKindInterface, nil))
		}
		for _, section := range entry.Actor.Sections {
			if section.Auth == nil {
				continue
			}
			if section.Auth.Credential != nil {
				children = append(children, sectionSymbol(source, "credential", section.Auth.Credential.Pos, dataMemberSymbols(source, section.Auth.Credential.Members)))
			}
			if section.Auth.Info != nil {
				children = append(children, sectionSymbol(source, "info", section.Auth.Info.Pos, dataMemberSymbols(source, section.Auth.Info.Members)))
			}
		}
	case entry.Resource != nil:
		for _, section := range entry.Resource.Sections {
			if section.Check != nil {
				children = append(children, resourceCheckSymbol(source, section.Check))
			}
			if section.Action != nil {
				actionChildren := make([]_Symbol, 0, len(section.Action.Checks))
				for _, check := range section.Action.Checks {
					actionChildren = append(actionChildren, resourceCheckSymbol(source, check))
				}
				children = append(children, newSymbol(source, section.Action.Name, "action", descriptionFromDecorators(section.Action.Decorators), protocol.SymbolKindMethod, actionChildren))
			}
		}
	case entry.Service != nil:
		for _, section := range entry.Service.Sections {
			if section.Method == nil {
				continue
			}
			method := section.Method
			methodChildren := []_Symbol{}
			if method.Input != nil {
				methodChildren = argumentSymbols(source, method.Input.Arguments)
			}
			children = append(children, newSymbol(source, method.Name, "method", descriptionFromDecoratorGroups(section.Decorators, method.Decorators), protocol.SymbolKindMethod, methodChildren))
		}
	case entry.Event != nil:
		if entry.Event.Payload != nil {
			children = dataMemberSymbols(source, entry.Event.Payload.Members)
		}
	case entry.Task != nil:
		for _, trigger := range entry.Task.Triggers {
			triggerChildren := []_Symbol{}
			if trigger.Input != nil {
				triggerChildren = argumentSymbols(source, trigger.Input.Arguments)
			}
			children = append(children, newSymbol(source, trigger.Name, "trigger", descriptionFromDecorators(trigger.Decorators), protocol.SymbolKindEvent, triggerChildren))
		}
	}
	return finishSymbol(_Symbol{Name: name, Detail: detail, Description: description, Kind: kind, Range: range_, Children: children})
}

func dataMemberSymbols(source string, members []*grammar.DataMember) []_Symbol {
	symbols := make([]_Symbol, 0, len(members))
	for _, member := range members {
		symbols = append(symbols, newSymbol(source, member.Name, "field", descriptionFromDecorators(member.Decorators), protocol.SymbolKindField, nil))
	}
	return symbols
}

func argumentSymbols(source string, arguments []*grammar.Argument) []_Symbol {
	symbols := make([]_Symbol, 0, len(arguments))
	for _, argument := range arguments {
		symbols = append(symbols, newSymbol(source, argument.Name, "parameter", descriptionFromDecorators(argument.Decorators), protocol.SymbolKindVariable, nil))
	}
	return symbols
}

func resourceCheckSymbol(source string, check *grammar.ResourceCheck) _Symbol {
	children := argumentSymbols(source, check.Arguments)
	return newSymbol(source, check.Name, "check", "", protocol.SymbolKindFunction, children)
}

func newSymbol(source string, name *grammar.Identifier, detail, description string, kind protocol.SymbolKind, children []_Symbol) _Symbol {
	range_ := identifierRange(source, name.Pos, name.Value)
	return finishSymbol(_Symbol{Name: name.Value, Detail: detail, Description: description, Kind: kind, Range: range_, Children: children})
}

func sectionSymbol(source, name string, pos lexer.Position, children []_Symbol) _Symbol {
	range_ := identifierRange(source, pos, name)
	return finishSymbol(_Symbol{Name: name, Detail: "actor auth section", Kind: protocol.SymbolKindObject, Range: range_, Children: children})
}

func finishSymbol(symbol _Symbol) _Symbol {
	for _, child := range symbol.Children {
		if comparePosition(child.Range.End, symbol.Range.End) > 0 {
			symbol.Range.End = child.Range.End
		}
	}
	return symbol
}

func descriptionFromDecorators(decorators []*grammar.Decorator) string {
	for _, decorator := range decorators {
		if decorator.Name.Value == "desc" && decorator.Value != nil {
			description, _ := grammar.UnquoteDescriptionString(decorator.Value.Raw)
			return description
		}
	}
	return ""
}

func descriptionFromDecoratorGroups(groups ...[]*grammar.Decorator) string {
	for _, decorators := range groups {
		if description := descriptionFromDecorators(decorators); description != "" {
			return description
		}
	}
	return ""
}
