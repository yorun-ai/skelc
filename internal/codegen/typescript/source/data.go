package source

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

const dataTsFilename = "data.ts"

//go:embed tpl/data.ts.tpl
var dataTsTemplate string

type DataTsPayload struct {
	TypeImports []*TypeImport
	Enums       []*Enum
	Data        []*Data
}

func (g *_Gen) genDataTs() {
	payload := g.buildDataTsPayload()
	g.renderTs(dataTsFilename, dataTsTemplate, payload)
}

func (g *_Gen) buildDataTsPayload() *DataTsPayload {
	if g.pubOnly {
		return g.buildPubDataTsPayload()
	}

	dataList := g.domain.Data()
	payload := &DataTsPayload{
		TypeImports: buildDataExternalImports(dataList),
		Enums:       sliceutil.Map(g.domain.Enums(), castEnum),
		Data:        make([]*Data, 0, len(dataList)),
	}
	for _, dataType := range dataList {
		castedData := castData(dataType)
		payload.Data = append(payload.Data, castedData)
	}
	return payload
}

func (g *_Gen) buildPubDataTsPayload() *DataTsPayload {
	dataList := g.publicView.Data
	payload := &DataTsPayload{
		TypeImports: buildDataExternalImports(dataList),
		Enums:       sliceutil.Map(g.publicView.Enums, castEnum),
		Data:        make([]*Data, 0, len(dataList)),
	}
	for _, dataType := range dataList {
		castedData := castData(dataType)
		payload.Data = append(payload.Data, castedData)
	}
	return payload
}

func buildDataExternalImports(dataList []*model.Data) []*TypeImport {
	imports := make([]*TypeImport, 0)
	seen := make(map[string]struct{})
	for _, dataType := range dataList {
		for _, member := range dataType.Members {
			imports = appendExternalTypeImports(imports, seen, member.Type)
		}
	}
	return imports
}

func appendExternalTypeImports(imports []*TypeImport, seen map[string]struct{}, type_ *model.Type) []*TypeImport {
	_ = common.WalkType(type_, func(current *model.Type) error {
		if current.ExternalImportPath != "" {
			key := current.ExternalAlias + "\x00" + current.ExternalImportPath
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				imports = append(imports, &TypeImport{Alias: current.ExternalAlias, Path: current.ExternalImportPath})
			}
		}
		return nil
	})
	sort.Slice(imports, func(i, j int) bool {
		if imports[i].Path == imports[j].Path {
			return imports[i].Alias < imports[j].Alias
		}
		return imports[i].Path < imports[j].Path
	})
	return imports
}

type Data struct {
	Name         string
	FullName     string
	CommentLines []string
	Members      []*DataMember
}

func castData(p *model.Data) *Data {
	data := &Data{
		Name:         transDataName(p),
		CommentLines: tsCommentLines(p.Description, ""),
		Members:      make([]*DataMember, 0, len(p.Members)),
	}
	for _, member := range p.Members {
		castedMember := castDataMember(member)
		data.Members = append(data.Members, castedMember)
	}

	data.FullName = data.Name
	if p.TypeParameters != nil {
		tpNames := sliceutil.Map(p.TypeParameters, func(tp *model.TypeParameter) string {
			return tp.Name
		})
		data.FullName = fmt.Sprintf("%s<%s>", data.Name, strings.Join(tpNames, ", "))
	}

	if len(data.Members) > 0 {
		memberNames := sliceutil.Map(data.Members, func(m *DataMember) string { return m.Name })
		maxMemberNameLen := nameutil.MaxLength(memberNames)
		sliceutil.ForEach(data.Members, func(m *DataMember) {
			m.NamePadding = nameutil.PaddingSpaces(maxMemberNameLen - len(m.Name))
		})
	}

	return data
}

func transDataName(p *model.Data) string {
	return nameutil.ToCamel(p.Name)
}

type DataMember struct {
	Name         string
	NamePadding  string
	CommentLines []string
	Type         *Type
}

func castDataMember(p *model.DataMember) *DataMember {
	memberType := castType(p.Type)
	return &DataMember{
		Name:         p.Name,
		CommentLines: tsCommentLines(p.Description, p.Example),
		Type:         memberType,
	}
}
