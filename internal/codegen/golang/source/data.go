package source

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

const dataGoFilename = "data.go"

var dataGoTemplate = loadGoTemplate("data.go.tpl")

type DataGoPayload struct {
	PackageName   string
	StdImports    []*Import
	ModuleImports []*Import
	Data          []*Data
}

func (g *_Gen) genDataGo() {
	payload := g.buildDataGoPayload()
	if len(payload.Data) > 0 {
		g.renderGo(dataGoFilename, dataGoTemplate, payload)
	}
}

func (g *_Gen) buildDataGoPayload() *DataGoPayload {
	payload := &DataGoPayload{
		PackageName: g.pkgName,
		Data:        make([]*Data, 0, len(g.view.Data)),
	}
	for _, dataType := range g.view.Data {
		castedData := castData(dataType)
		payload.Data = append(payload.Data, castedData)
	}
	imports := buildDataImports(payload.Data)
	payload.StdImports, payload.ModuleImports = splitImports(imports)

	return payload
}

type Data struct {
	Name            string
	FullName        string
	ReceiverType    string
	ImplName        string
	ConstructorName string
	SpecName        string
	SkelName        string
	Hash            string
	Lifecycle       string
	RegisterFunc    string
	CommentLines    []string
	Members         []*DataMember
	Validate        bool
	CheckLines      []string
}

func castData(p *model.Data) *Data {
	data := &Data{
		Name:            transDataName(p),
		ImplName:        "_" + transDataName(p),
		ConstructorName: "_New" + transDataName(p),
		CommentLines:    goDocLines(transDataName(p), p.Description),
		Members:         make([]*DataMember, 0, len(p.Members)),
	}
	for _, member := range p.Members {
		castedMember := castDataMember(member)
		data.Members = append(data.Members, castedMember)
	}

	data.FullName = data.Name
	data.ReceiverType = data.Name
	if p.TypeParameters != nil {
		tpNames := sliceutil.Map(p.TypeParameters, func(tp *model.TypeParameter) string {
			return tp.Name
		})
		data.FullName = fmt.Sprintf("%s[%s any]", data.Name, strings.Join(tpNames, ", "))
		data.ReceiverType = fmt.Sprintf("%s[%s]", data.Name, strings.Join(tpNames, ", "))
	}
	data.Validate = dataNeedsCheck(p, map[*model.Data]bool{})
	if data.Validate {
		data.CheckLines = buildDataCheckLines(p)
	}

	return data
}

func transDataName(p *model.Data) string {
	return nameutil.ToCamel(p.Name)
}

type DataMember struct {
	Name         string
	CommentLines []string
	Type         *Type
	SkelName     string
}

func castDataMember(p *model.DataMember) *DataMember {
	memberType := castType(p.Type)
	return &DataMember{
		Name:         nameutil.ToCamel(p.Name),
		CommentLines: goDocLines(nameutil.ToCamel(p.Name), common.MergeDescriptionAndExample(p.Description, p.Example)),
		Type:         memberType,
		SkelName:     p.Name,
	}
}

func buildDataImports(dataList []*Data) []*Import {
	imports := newImportSet()
	for _, data := range dataList {
		if data.Validate {
			imports.add(&Import{Path: "go.yorun.ai/vine/core/rpc"})
		}
		for _, member := range data.Members {
			imports.addMany(collectTypeImports(member.Type))
		}
	}
	return imports.sortedValues()
}

func dataNeedsCheck(p *model.Data, visiting map[*model.Data]bool) bool {
	if p == nil || visiting[p] {
		return false
	}
	visiting[p] = true
	defer delete(visiting, p)
	for _, member := range p.Members {
		if typeNeedsCheck(member.Type, visiting) {
			return true
		}
	}
	return false
}

func typeNeedsCheck(type_ *model.Type, visiting map[*model.Data]bool) bool {
	if type_ == nil {
		return false
	}
	switch type_.Kind {
	case model.TypeKindList:
		return !type_.Nullable || typeNeedsCheck(type_.List.Value, visiting)
	case model.TypeKindMap:
		return !type_.Nullable || typeNeedsCheck(type_.Map.Value, visiting)
	case model.TypeKindData:
		return dataNeedsCheck(type_.Data, visiting)
	default:
		return false
	}
}

func buildDataCheckLines(p *model.Data) []string {
	lines := []string{}
	for _, member := range p.Members {
		memberName := nameutil.ToCamel(member.Name)
		lines = append(lines, buildTypeCheckLines(member.Type, "v."+memberName, fmt.Sprintf("rpc.JoinPath(path, %q)", memberName), "\t", 0)...)
	}
	return lines
}

func buildTypeCheckLines(type_ *model.Type, expr string, pathExpr string, indent string, depth int) []string {
	if type_ == nil {
		return nil
	}
	lines := []string{}
	switch type_.Kind {
	case model.TypeKindList:
		if !type_.Nullable {
			lines = append(lines,
				fmt.Sprintf("%sif err := rpc.CheckValueNotNil(%s, %s); err != nil {", indent, expr, pathExpr),
				fmt.Sprintf("%s\treturn err", indent),
				fmt.Sprintf("%s}", indent),
			)
		}
		if typeNeedsCheck(type_.List.Value, map[*model.Data]bool{}) {
			indexName := fmt.Sprintf("i%d", depth)
			lines = append(lines, fmt.Sprintf("%sfor %s := range %s {", indent, indexName, expr))
			lines = append(lines, buildTypeCheckLines(type_.List.Value, fmt.Sprintf("%s[%s]", expr, indexName), fmt.Sprintf("rpc.JoinIndex(%s, %s)", pathExpr, indexName), indent+"\t", depth+1)...)
			lines = append(lines, fmt.Sprintf("%s}", indent))
		}
	case model.TypeKindMap:
		if !type_.Nullable {
			lines = append(lines,
				fmt.Sprintf("%sif err := rpc.CheckValueNotNil(%s, %s); err != nil {", indent, expr, pathExpr),
				fmt.Sprintf("%s\treturn err", indent),
				fmt.Sprintf("%s}", indent),
			)
		}
		if typeNeedsCheck(type_.Map.Value, map[*model.Data]bool{}) {
			keyName := fmt.Sprintf("key%d", depth)
			itemName := fmt.Sprintf("item%d", depth)
			lines = append(lines, fmt.Sprintf("%sfor %s, %s := range %s {", indent, keyName, itemName, expr))
			lines = append(lines, buildTypeCheckLines(type_.Map.Value, itemName, fmt.Sprintf("rpc.JoinMapKey(%s, %s)", pathExpr, keyName), indent+"\t", depth+1)...)
			lines = append(lines, fmt.Sprintf("%s}", indent))
		}
	case model.TypeKindData:
		if !dataNeedsCheck(type_.Data, map[*model.Data]bool{}) {
			return lines
		}
		if type_.Nullable {
			lines = append(lines,
				fmt.Sprintf("%sif %s != nil {", indent, expr),
				fmt.Sprintf("%s\tif err := %s.Validate(%s); err != nil {", indent, expr, pathExpr),
				fmt.Sprintf("%s\t\treturn err", indent),
				fmt.Sprintf("%s\t}", indent),
				fmt.Sprintf("%s}", indent),
			)
			return lines
		}
		lines = append(lines,
			fmt.Sprintf("%sif err := (&%s).Validate(%s); err != nil {", indent, expr, pathExpr),
			fmt.Sprintf("%s\treturn err", indent),
			fmt.Sprintf("%s}", indent),
		)
	}
	return lines
}
