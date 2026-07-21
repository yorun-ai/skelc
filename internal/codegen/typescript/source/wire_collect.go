package source

import (
	"sort"

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
	if type_ == nil {
		return
	}

	switch type_.Kind {
	case model.TypeKindList:
		b.collectType(type_.List.Value)
	case model.TypeKindMap:
		b.collectType(type_.Map.Value)
	case model.TypeKindData:
		for _, argument := range type_.TypeArguments {
			b.collectType(argument)
		}
		if b.data[type_.Data] {
			return
		}
		b.data[type_.Data] = true
		for _, member := range type_.Data.Members {
			b.collectType(member.Type)
		}
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
