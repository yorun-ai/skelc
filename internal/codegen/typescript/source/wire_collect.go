package source

import (
	"sort"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/model"
)

func (b *_WireSchemaBuilder) collectMethod(method *model.Method) {
	if methodArgumentsContainBinary(method) {
		for _, argument := range method.Arguments {
			b.collectType(argument.Type)
		}
	}
	if methodResultContainsBinary(method) {
		b.collectType(method.ResultType)
	}
}

func (b *_WireSchemaBuilder) collectType(type_ *model.Type) {
	if err := common.WalkTypeGraph(type_, func(current *model.Type) error {
		if current.Kind == model.TypeKindData {
			b.data[current.Data] = true
		}
		return nil
	}); err != nil {
		b.fail("collect TypeScript wire schema types: %v", err)
	}
}

func (b *_WireSchemaBuilder) prepareFactoryNames() {
	dataList := b.sortedData()
	nameCounts := map[string]int{}
	for _, dataType := range dataList {
		nameCounts[dataType.Name]++
	}

	for _, dataType := range dataList {
		name := dataType.Name
		if nameCounts[name] > 1 {
			name = dataType.SkelName
			if name == "" {
				name = dataType.Domain + "." + dataType.Name
			}
		}
		b.factoryNames[dataType] = "create" + nameutil.ToCamel(name) + "WireSchema"
	}
}

func (b *_WireSchemaBuilder) sortedData() []*model.Data {
	dataList := make([]*model.Data, 0, len(b.data))
	for dataType := range b.data {
		dataList = append(dataList, dataType)
	}
	sort.Slice(dataList, func(i, j int) bool {
		left := dataList[i].SkelName
		if left == "" {
			left = dataList[i].Domain + "." + dataList[i].Name
		}
		right := dataList[j].SkelName
		if right == "" {
			right = dataList[j].Domain + "." + dataList[j].Name
		}
		return left < right
	})
	return dataList
}
