package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

type Enum struct {
	Name         string
	CommentLines []string
	Items        []*EnumItem
}

func castEnum(p *model.Enum) *Enum {
	enum := &Enum{
		Name:         nameutil.ToCamel(p.Name),
		CommentLines: tsCommentLines(p.Description, ""),
		Items:        make([]*EnumItem, 0, len(p.Items)+1),
	}
	enum.Items = append(enum.Items, castEnumItem(p.UnspecifiedItem))
	enum.Items = append(enum.Items, sliceutil.Map(p.Items, castEnumItem)...)
	if len(enum.Items) > 0 {
		itemValues := sliceutil.Map(enum.Items, func(i *EnumItem) string { return i.Literal })
		maxValueLen := nameutil.MaxLength(itemValues)
		sliceutil.ForEach(enum.Items, func(i *EnumItem) {
			i.ValuePadding = nameutil.PaddingSpaces(maxValueLen - len(i.Literal))
		})
	}

	return enum
}

func transEnumName(p *model.Enum) string {
	return nameutil.ToCamel(p.Name)
}

type EnumItem struct {
	Literal      string
	Value        string
	ValuePadding string
	CommentLines []string
}

func castEnumItem(p *model.EnumItem) *EnumItem {
	return &EnumItem{
		Literal:      fmt.Sprintf(`"%s"`, nameutil.ToScreamingSnake(p.Name)),
		Value:        nameutil.ToScreamingSnake(p.Name),
		CommentLines: tsCommentLines(common.MergeDescriptionAndExample(p.Description, ""), ""),
	}
}
