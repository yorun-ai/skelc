package analyzer

import (
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

const unspecifiedEnumName = "UNSPECIFIED"

func parseEnum(reporter *diagnosticReporter, ge *grammar.Enum) (*model.Enum, bool) {
	valid := checkCase(reporter, "Enum", caseTypeCamel, ge.Name)
	valid = checkNotReservedKindSuffix(reporter, "Enum", ge.Name) && valid

	enum := &model.Enum{
		Pos:  position(ge.Name.Pos),
		Name: ge.Name.Value,
		Pub:  ge.Pub,
		UnspecifiedItem: &model.EnumItem{
			Name: unspecifiedEnumName,
		},
		Items: []*model.EnumItem{},
	}
	meta, metaValid := parseDecoratorMeta(reporter, ge.Decorators, decoratorContext{
		allowDesc: true,
	})
	valid = metaValid && valid
	enum.Description = meta.Description
	itemPositionByName := map[string]lexer.Position{}

	for _, grammarItem := range ge.Items {
		item, itemValid := parseEnumItem(reporter, grammarItem)
		valid = itemValid && valid
		duplicatedPosition, duplicated := itemPositionByName[item.Name]
		if duplicated {
			reporter.reportf("%s duplicated EnumItem %s found, also present at %s", item.Pos, item.Name, duplicatedPosition)
			valid = false
			continue
		}
		itemPositionByName[item.Name] = lexer.Position{Filename: item.Pos.File, Line: item.Pos.Line, Column: item.Pos.Column}
		enum.Items = append(enum.Items, item)
	}

	valid = reporter.check(len(enum.Items) > 0, "%s missing EnumItem for %s", enum.Pos, enum.Name) && valid
	return enum, valid
}

func parseEnumItem(reporter *diagnosticReporter, gei *grammar.EnumItem) (*model.EnumItem, bool) {
	valid := checkCase(reporter, "EnumItem", caseTypeScreamingSnake, gei.Name)
	valid = reporter.check(gei.Name.Value != unspecifiedEnumName, "%s reversed EnumItem value %s", gei.Name.Pos, gei.Name.Value) && valid
	meta, metaValid := parseDecoratorMeta(reporter, gei.Decorators, decoratorContext{
		allowDesc: true,
	})
	valid = metaValid && valid

	return &model.EnumItem{
		Pos:         position(gei.Name.Pos),
		Name:        gei.Name.Value,
		Description: meta.Description,
	}, valid
}
