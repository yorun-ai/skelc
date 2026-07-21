package source

import (
	"strings"

	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

const enumGoFilename = "enum.go"

var enumGoTemplate = loadGoTemplate("enum.go.tpl")

type EnumGoPayload struct {
	PackageName   string
	StdImports    []*Import
	ModuleImports []*Import
	Enums         []*Enum
}

func (g *_Gen) genEnumGo() {
	payload := g.buildEnumGoPayload()
	if len(payload.Enums) > 0 {
		g.renderGo(enumGoFilename, enumGoTemplate, payload)
	}
}

func (g *_Gen) buildEnumGoPayload() *EnumGoPayload {
	p := &EnumGoPayload{
		PackageName: g.pkgName,
		Enums:       sliceutil.Map(g.view.Enums, castEnum),
	}
	if len(p.Enums) > 0 {
		imports := []*Import{{Path: "fmt"}}
		p.StdImports, p.ModuleImports = splitImports(imports)
	}
	return p
}

type Enum struct {
	Name            string
	VarName         string
	CommentLines    []string
	UnspecifiedItem *EnumItem
	Items           []*EnumItem
}

func castEnum(p *model.Enum) *Enum {
	enum := &Enum{
		Name:            transEnumName(p),
		VarName:         nameutil.ToLowerCamel(p.Name),
		UnspecifiedItem: castEnumItem(p.UnspecifiedItem),
		Items:           sliceutil.Map(p.Items, castEnumItem),
	}
	enum.CommentLines = goDocLines(enum.Name, p.Description)

	if len(enum.Items) > 0 {
		enum.UnspecifiedItem.Name = enum.Name + enum.UnspecifiedItem.Name
		enum.UnspecifiedItem.CommentLines = goDocLines(enum.UnspecifiedItem.Name, codegen.MergeDescriptionAndExample(p.UnspecifiedItem.Description, ""))
		sliceutil.ForEach(enum.Items, func(i *EnumItem) {
			i.Name = enum.Name + i.Name
			i.CommentLines = goDocLines(i.Name, codegen.MergeDescriptionAndExample(i.Description, ""))
		})
	}

	return enum
}

func transEnumName(p *model.Enum) string {
	return nameutil.ToCamel(p.Name)
}

func transUnspecifiedItemName(p *model.Enum) string {
	return transEnumName(p) + castEnumItem(p.UnspecifiedItem).Name
}

type EnumItem struct {
	Name         string
	Value        string
	Description  string
	CommentLines []string
}

func castEnumItem(p *model.EnumItem) *EnumItem {
	name := nameutil.ToCamel(strings.ToLower(p.Name))
	return &EnumItem{
		Name:         name,
		Value:        nameutil.ToScreamingSnake(p.Name),
		Description:  p.Description,
		CommentLines: goDocLines(name, codegen.MergeDescriptionAndExample(p.Description, "")),
	}
}
