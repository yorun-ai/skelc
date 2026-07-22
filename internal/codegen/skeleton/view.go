package skeleton

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

type _PubView struct {
	enums     []*model.Enum
	dataList  []*model.Data
	configs   []*model.Data
	actors    []*model.Actor
	resources []*model.Resource
	events    []*model.Data
	services  []*model.Service
}

func newPubView(domain *model.Domain) *_PubView {
	actors := filterPubActors(domain.Actors())
	return &_PubView{
		enums:     filterPubEnums(domain.Enums()),
		dataList:  filterPubData(domain.Data()),
		configs:   filterPubData(domain.Configs()),
		actors:    actors,
		resources: filterPubResources(domain.Resources()),
		events:    filterPubData(domain.Events()),
		services:  filterPubServices(domain.Services()),
	}
}

func filterPubEnums(enums []*model.Enum) []*model.Enum {
	filtered := make([]*model.Enum, 0, len(enums))
	for _, enum := range enums {
		if enum.Pub {
			filtered = append(filtered, enum)
		}
	}
	return filtered
}

func filterPubData(dataList []*model.Data) []*model.Data {
	filtered := make([]*model.Data, 0, len(dataList))
	for _, data := range dataList {
		if data.Pub {
			filtered = append(filtered, data)
		}
	}
	return filtered
}

func filterPubActors(actors []*model.Actor) []*model.Actor {
	filtered := make([]*model.Actor, 0, len(actors))
	for _, actor := range actors {
		if actor.Pub {
			filtered = append(filtered, actor)
		}
	}
	return filtered
}

func filterPubServices(services []*model.Service) []*model.Service {
	filtered := make([]*model.Service, 0, len(services))
	for _, service := range services {
		if service.Pub {
			filtered = append(filtered, service)
		}
	}
	return filtered
}

func filterPubResources(resources []*model.Resource) []*model.Resource {
	filtered := make([]*model.Resource, 0, len(resources))
	for _, resource := range resources {
		if resource.Pub {
			filtered = append(filtered, resource)
		}
	}
	return filtered
}

func validatePubView(domain *model.Domain, view *_PubView) {
	pubResourceBySkelName := map[string]struct{}{}
	for _, resource := range view.resources {
		pubResourceBySkelName[resource.SkelName] = struct{}{}
	}
	for _, data := range view.dataList {
		validatePubMembers(fmt.Sprintf("pub data %s", data.Name), data.Members, map[*model.Data]struct{}{})
	}
	for _, config := range view.configs {
		validatePubMembers(fmt.Sprintf("pub config %s", config.Name), config.Members, map[*model.Data]struct{}{})
	}
	for _, actor := range view.actors {
		if actor.AuthEnabled {
			validatePubMembers(fmt.Sprintf("pub actor %s credential", actor.Name), actor.AuthCredential.Members, map[*model.Data]struct{}{})
			validatePubMembers(fmt.Sprintf("pub actor %s info", actor.Name), actor.AuthInfo.Members, map[*model.Data]struct{}{})
		}
	}
	for _, service := range view.services {
		for _, audience := range service.Audiences {
			if actor, ok := findActor(domain.Actors(), audience.Actor); ok {
				checkutil.Check(actor.Pub, "pub service %s references non-pub actor %s", service.Name, actor.Name)
			}
		}
		validatePubRequire(domain.Name(), fmt.Sprintf("pub service %s", service.Name), service.Require, pubResourceBySkelName)
		for _, method := range service.Methods {
			context := fmt.Sprintf("pub service %s.%s", service.Name, method.Name)
			validatePubRequire(domain.Name(), context, method.Require, pubResourceBySkelName)
			for _, argument := range method.Arguments {
				validatePubType(context, argument.Type, map[*model.Data]struct{}{})
			}
			validatePubType(context, method.ResultType, map[*model.Data]struct{}{})
		}
	}
	for _, event := range view.events {
		validatePubMembers(fmt.Sprintf("pub event %s", event.Name), event.Members, map[*model.Data]struct{}{})
	}
}

func validatePubRequire(domainName string, context string, require *model.PermissionRequire, pubResourceBySkelName map[string]struct{}) {
	if require == nil {
		return
	}
	validatePubRequireExpr(domainName, context, require.Expr, pubResourceBySkelName)
}

func validatePubRequireExpr(domainName string, context string, expr *model.PermissionExpr, pubResourceBySkelName map[string]struct{}) {
	if expr.Code != "" {
		resourceSkelName := permissionCodeResourceSkelName(expr.Code)
		_, pubResource := pubResourceBySkelName[resourceSkelName]
		if isLocalResourceSkelName(domainName, resourceSkelName) && !pubResource {
			checkutil.Failf("%s references non-pub resource %s", context, resourceSkelName)
		}
	}
	for _, child := range expr.Children {
		validatePubRequireExpr(domainName, context, child, pubResourceBySkelName)
	}
}

func isLocalResourceSkelName(domainName string, resourceSkelName string) bool {
	return strings.HasPrefix(resourceSkelName, domainName+".")
}

func permissionCodeResourceSkelName(code string) string {
	index := strings.LastIndex(code, ":")
	checkutil.Check(index > 0, "invalid permission code %s", code)
	return code[:index]
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
