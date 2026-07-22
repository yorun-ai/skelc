package analyzer

import (
	"fmt"

	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/graphutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
	"strings"
)

func (p *Analysis) checkHardCycleReferences() {
	graph := graphutil.New[*model.Data]()
	refs := _RefsMatrix{}

	for _, dataType := range p.dataMap {
		if refs.has(dataType) {
			continue
		}

		refs[dataType] = _Refs{}
		for _, member := range dataType.Members {
			refs[dataType].merge(referencedData(member.Type))
		}
		for refData := range refs[dataType] {
			graph.AddEdge(dataType, refData)
		}
	}

	for _, cycle := range graph.FindCycles() {
		cycle = append(cycle, cycle[0])
		isHard := true

		for si, di := 0, 1; di < len(cycle); si, di = si+1, di+1 {
			if !refs.refKind(cycle[si], cycle[di]).isHard() {
				isHard = false
				break
			}
		}

		checkutil.CheckFuncAt(cycle[0].Pos, !isHard, func() string {
			names := sliceutil.Map(cycle, func(dataType *model.Data) string {
				return dataType.Name
			})
			return fmt.Sprintf("hard reference chain detected: %s, try nullable/list/map instead", strings.Join(names, " -> "))
		})
	}
}

func (p *Analysis) checkDataDoesNotReferenceConfigs(dataList []*model.Data) {
	for _, dataType := range dataList {
		if dataType.Kind != model.DataKindData {
			continue
		}
		for _, member := range dataType.Members {
			for refData := range referencedData(member.Type) {
				checkutil.Check(refData.Kind != model.DataKindConfig,
					"%s data %s cannot reference config %s", member.Pos, dataType.Name, refData.Name)
			}
		}
	}
}
