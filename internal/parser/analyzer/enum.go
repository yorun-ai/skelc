package analyzer

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

const unspecifiedEnumName = "UNSPECIFIED"

func parseEnum(ge *grammar.Enum) *model.Enum {
	checkCase("Enum", caseTypeCamel, ge.Name)
	checkNotReservedKindSuffix("Enum", ge.Name)

	enum := &model.Enum{
		Pos:  position(ge.Name.Pos),
		Name: ge.Name.Value,
		Pub:  ge.Pub,
		UnspecifiedItem: &model.EnumItem{
			Name: unspecifiedEnumName,
		},
		Items: []*model.EnumItem{},
	}
	meta := parseDecoratorMeta(ge.Decorators, decoratorContext{
		allowDesc: true,
	})
	enum.Description = meta.Description
	itemPositionByName := map[string]lexer.Position{}

	for _, grammarItem := range ge.Items {
		item := parseEnumItem(grammarItem)
		duplicatedPosition, duplicated := itemPositionByName[item.Name]
		checkutil.CheckFuncAt(item.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated EnumItem %s found, also present at %s",
				item.Pos, item.Name, duplicatedPosition)
		})
		itemPositionByName[item.Name] = lexer.Position{Filename: item.Pos.File, Line: item.Pos.Line, Column: item.Pos.Column}
		enum.Items = append(enum.Items, item)
	}

	checkutil.Check(len(enum.Items) > 0, "%s missing EnumItem for %s", enum.Pos, enum.Name)
	return enum
}

func parseEnumItem(gei *grammar.EnumItem) *model.EnumItem {
	checkCase("EnumItem", caseTypeScreamingSnake, gei.Name)
	checkutil.Check(gei.Name.Value != unspecifiedEnumName, "%s reversed EnumItem value %s", gei.Name.Pos, gei.Name.Value)
	meta := parseDecoratorMeta(gei.Decorators, decoratorContext{
		allowDesc: true,
	})

	return &model.EnumItem{
		Pos:         position(gei.Name.Pos),
		Name:        gei.Name.Value,
		Description: meta.Description,
	}
}
