package source

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/skelmeta"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

const dataGoFilename = "data.go"

var dataGoTemplate = joinTemplates("imports.go.tpl", "go_ir.go.tpl", "data_clone.go.tpl", "data.go.tpl")

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
		castedData := castCloneableData(dataType)
		payload.Data = append(payload.Data, castedData)
	}
	imports := buildDataImports(payload.Data)
	payload.StdImports, payload.ModuleImports = splitImports(imports)

	return payload
}

type Data struct {
	Name             string
	FullName         string
	ReceiverType     string
	ImplName         string
	ConstructorName  string
	SpecName         string
	SkelName         string
	Hash             string
	Lifecycle        string
	RegisterFunc     string
	CommentLines     []string
	Members          []*DataMember
	Sensitive        bool
	MarkerMethodName string
	Validate         bool
	CheckBlock       *_GoBlock
	Clone            bool
	CloneMethodName  string
	CloneParameters  []*_GoParameter
	CloneBlock       *_GoBlock
	CloneImports     []*Import
}

func castData(p *model.Data) *Data {
	data := &Data{
		Name:             transDataName(p),
		ImplName:         "_" + transDataName(p),
		ConstructorName:  "_New" + transDataName(p),
		CommentLines:     deprecatedGoDocLines(goDocLines(transDataName(p), p.Description), transDataName(p), p.DeprecatedReason),
		Members:          make([]*DataMember, 0, len(p.Members)),
		Sensitive:        p.Sensitive,
		MarkerMethodName: skelmeta.SensitiveMarkerMethodName,
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
		data.CheckBlock = buildDataCheckBlock(p)
	}
	return data
}

func castCloneableData(p *model.Data) *Data {
	data := castData(p)
	buildDataClone(p, data)
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
	Sensitive    bool
}

func castDataMember(p *model.DataMember) *DataMember {
	memberType := castType(p.Type)
	return &DataMember{
		Name: nameutil.ToCamel(p.Name),
		CommentLines: deprecatedGoDocLines(
			goDocLines(nameutil.ToCamel(p.Name), common.MergeDescriptionAndExample(p.Description, p.Example)),
			nameutil.ToCamel(p.Name),
			p.DeprecatedReason,
		),
		Type:      memberType,
		SkelName:  p.Name,
		Sensitive: p.Sensitive,
	}
}

func buildDataImports(dataList []*Data) []*Import {
	imports := newImportSet()
	for _, data := range dataList {
		if data.Validate {
			imports.add(&Import{Path: "go.yorun.ai/vine/core/rpc"})
		}
		imports.addMany(data.CloneImports)
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

func buildDataCheckBlock(p *model.Data) *_GoBlock {
	block := goBlock()
	for _, member := range p.Members {
		memberName := nameutil.ToCamel(member.Name)
		block.append(buildTypeCheckStatements(
			member.Type,
			"v."+memberName,
			fmt.Sprintf("rpc.JoinPath(path, %q)", memberName),
			0,
		)...)
	}
	return block
}

func buildTypeCheckStatements(type_ *model.Type, expr string, pathExpr string, depth int) []*_GoStatement {
	if type_ == nil {
		return nil
	}
	statements := []*_GoStatement{}
	switch type_.Kind {
	case model.TypeKindList:
		if !type_.Nullable {
			statements = append(statements, buildCheckValueNotNilStatement(expr, pathExpr))
		}
		if typeNeedsCheck(type_.List.Value, map[*model.Data]bool{}) {
			indexName := fmt.Sprintf("i%d", depth)
			statements = append(statements, goRangeStatement(
				[]string{indexName},
				goRaw(expr),
				goBlock(buildTypeCheckStatements(
					type_.List.Value,
					fmt.Sprintf("%s[%s]", expr, indexName),
					fmt.Sprintf("rpc.JoinIndex(%s, %s)", pathExpr, indexName),
					depth+1,
				)...),
			))
		}
	case model.TypeKindMap:
		if !type_.Nullable {
			statements = append(statements, buildCheckValueNotNilStatement(expr, pathExpr))
		}
		if typeNeedsCheck(type_.Map.Value, map[*model.Data]bool{}) {
			keyName := fmt.Sprintf("key%d", depth)
			itemName := fmt.Sprintf("item%d", depth)
			statements = append(statements, goRangeStatement(
				[]string{keyName, itemName},
				goRaw(expr),
				goBlock(buildTypeCheckStatements(
					type_.Map.Value,
					itemName,
					fmt.Sprintf("rpc.JoinMapKey(%s, %s)", pathExpr, keyName),
					depth+1,
				)...),
			))
		}
	case model.TypeKindData:
		if !dataNeedsCheck(type_.Data, map[*model.Data]bool{}) {
			return statements
		}
		if type_.Nullable {
			statements = append(statements, goIfStatement(
				nil,
				goRaw(expr+" != nil"),
				goBlock(buildValidateStatement(expr, pathExpr)),
				nil,
			))
			return statements
		}
		statements = append(statements, buildValidateStatement("(&"+expr+")", pathExpr))
	}
	return statements
}

func buildCheckValueNotNilStatement(expr string, pathExpr string) *_GoStatement {
	return goIfStatement(
		goAssignment("err", ":=", goCall("rpc.CheckValueNotNil", goRaw(expr), goRaw(pathExpr))),
		goRaw("err != nil"),
		goBlock(goReturnStatement(goRaw("err"))),
		nil,
	)
}

func buildValidateStatement(expr string, pathExpr string) *_GoStatement {
	return goIfStatement(
		goAssignment("err", ":=", goCall(expr+".Validate", goRaw(pathExpr))),
		goRaw("err != nil"),
		goBlock(goReturnStatement(goRaw("err"))),
		nil,
	)
}
