package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

func pubEnums(enums []*model.Enum) []*model.Enum {
	filtered := make([]*model.Enum, 0, len(enums))
	for _, enum := range enums {
		if enum.Pub {
			filtered = append(filtered, enum)
		}
	}
	return filtered
}

func pubData(dataList []*model.Data) []*model.Data {
	filtered := make([]*model.Data, 0, len(dataList))
	for _, data := range dataList {
		if data.Pub {
			filtered = append(filtered, data)
		}
	}
	return filtered
}

func (g *_Gen) validatePubOnly() {
	if !g.pubOnly {
		return
	}

	for _, data := range g.domain.Data() {
		if !data.Pub {
			continue
		}
		validatePubMembers(fmt.Sprintf("pub data %s", data.Name), data.Members, map[*model.Data]struct{}{})
	}
	for _, service := range g.clientServices() {
		if !service.Pub {
			continue
		}
		for _, audience := range service.Audiences {
			if actor, ok := findActor(g.domain.Actors(), audience.Actor); ok {
				checkutil.Check(actor.Pub, "pub service %s references non-pub actor %s", service.Name, actor.Name)
			}
		}
		for _, method := range service.Methods {
			context := fmt.Sprintf("pub service %s.%s", service.Name, method.Name)
			for _, argument := range method.Arguments {
				validatePubType(context, argument.Type, map[*model.Data]struct{}{})
			}
			validatePubType(context, method.ResultType, map[*model.Data]struct{}{})
		}
	}
}

func validatePubMembers(context string, members []*model.DataMember, visitedData map[*model.Data]struct{}) {
	for _, member := range members {
		validatePubType(context, member.Type, visitedData)
	}
}

func validatePubType(context string, type_ *model.Type, visitedData map[*model.Data]struct{}) {
	if type_ == nil {
		return
	}

	switch type_.Kind {
	case model.TypeKindEnum:
		if type_.Enum != nil {
			checkutil.Check(type_.Enum.Pub, "%s references non-pub enum %s", context, type_.Enum.Name)
		}
	case model.TypeKindData:
		if type_.Data == nil {
			return
		}
		checkutil.Check(type_.Data.Kind != model.DataKindData || type_.Data.Pub,
			"%s references non-pub data %s", context, type_.Data.Name)
		if _, ok := visitedData[type_.Data]; ok {
			return
		}
		visitedData[type_.Data] = struct{}{}
		for _, typeArgument := range type_.TypeArguments {
			validatePubType(context, typeArgument, visitedData)
		}
		for _, member := range type_.Data.Members {
			validatePubType(context, member.Type, visitedData)
		}
	case model.TypeKindList:
		validatePubType(context, type_.List.Value, visitedData)
	case model.TypeKindMap:
		validatePubType(context, type_.Map.Key, visitedData)
		validatePubType(context, type_.Map.Value, visitedData)
	}
}

func findActor(actors []*model.Actor, name string) (*model.Actor, bool) {
	for _, actor := range actors {
		if actor.Name == name {
			return actor, true
		}
	}
	return nil, false
}
