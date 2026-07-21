package view

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

func validatePubView(domain *model.Domain, services []*model.Service, events []*model.Data, configs []*model.Data, resources []*model.Resource) {
	pubResourceBySkelName := map[string]struct{}{}
	for _, resource := range resources {
		pubResourceBySkelName[resource.SkelName] = struct{}{}
	}
	for _, data := range domain.Data() {
		if !data.Pub {
			continue
		}
		validatePubMembers(fmt.Sprintf("pub data %s", data.Name), data.Members, map[*model.Data]struct{}{})
	}
	for _, config := range configs {
		validatePubMembers(fmt.Sprintf("pub config %s", config.Name), config.Members, map[*model.Data]struct{}{})
	}
	for _, service := range services {
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
	for _, event := range events {
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
			checkutil.Panicf("%s references non-pub resource %s", context, resourceSkelName)
		}
	}
	for _, child := range expr.Children {
		validatePubRequireExpr(domainName, context, child, pubResourceBySkelName)
	}
}

func isLocalResourceSkelName(domainName string, resourceSkelName string) bool {
	return strings.HasPrefix(resourceSkelName, domainName+".")
}

func validatePubMembers(context string, members []*model.DataMember, visitedData map[*model.Data]struct{}) {
	for _, member := range members {
		validatePubType(context, member.Type, visitedData)
	}
}

func permissionCodeResourceSkelName(code string) string {
	index := strings.LastIndex(code, ":")
	checkutil.Check(index > 0, "invalid permission code %s", code)
	return code[:index]
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
