package skeleton

import (
	"strings"

	"go.yorun.ai/skelc/model"
)

func collectTypeImports(domain *model.Domain, view *_PubView) []*model.Import {
	used := map[string]struct{}{}
	for _, data := range view.dataList {
		collectImportsFromData(used, data)
	}
	for _, config := range view.configs {
		collectImportsFromData(used, config)
	}
	for _, resource := range view.resources {
		for _, check := range resource.Checks {
			for _, argument := range renderResourceCheckArguments(check) {
				collectImportsFromType(used, argument.Type)
			}
		}
		for _, action := range resource.Actions {
			for _, check := range action.Checks {
				for _, argument := range renderResourceCheckArguments(check) {
					collectImportsFromType(used, argument.Type)
				}
			}
		}
	}
	return selectUsedImports(domain.Imports(), used)
}

func collectDataImports(domain *model.Domain, dataList []*model.Data) []*model.Import {
	used := map[string]struct{}{}
	for _, data := range dataList {
		collectImportsFromData(used, data)
	}
	return selectUsedImports(domain.Imports(), used)
}

func collectActorImports(domain *model.Domain, actors []*model.Actor) []*model.Import {
	used := map[string]struct{}{}
	for _, actor := range actors {
		if actor.AuthEnabled {
			collectImportsFromData(used, actor.AuthCredential)
			collectImportsFromData(used, actor.AuthInfo)
		}
	}
	return selectUsedImports(domain.Imports(), used)
}

func collectServiceImports(domain *model.Domain, services []*model.Service) []*model.Import {
	used := map[string]struct{}{}
	for _, service := range services {
		for _, audience := range service.Audiences {
			collectImportFromQualifiedName(used, audience.Actor)
		}
		for _, method := range service.Methods {
			collectImportsFromType(used, method.ResultType)
			for _, argument := range method.Arguments {
				collectImportsFromType(used, argument.Type)
			}
		}
	}
	return selectUsedImports(domain.Imports(), used)
}

func collectImportFromQualifiedName(used map[string]struct{}, name string) {
	qualifier, _, ok := strings.Cut(name, ".")
	if ok {
		used[qualifier] = struct{}{}
	}
}

func collectImportsFromData(used map[string]struct{}, data *model.Data) {
	for _, member := range data.Members {
		collectImportsFromType(used, member.Type)
	}
}

func collectImportsFromType(used map[string]struct{}, type_ *model.Type) {
	if type_ == nil {
		return
	}
	if type_.ExternalDomain != "" {
		used[type_.ExternalDomain] = struct{}{}
	}
	switch type_.Kind {
	case model.TypeKindData:
		for _, typeArgument := range type_.TypeArguments {
			collectImportsFromType(used, typeArgument)
		}
	case model.TypeKindList:
		collectImportsFromType(used, type_.List.Value)
	case model.TypeKindMap:
		collectImportsFromType(used, type_.Map.Key)
		collectImportsFromType(used, type_.Map.Value)
	}
}

func selectUsedImports(imports []*model.Import, used map[string]struct{}) []*model.Import {
	selected := make([]*model.Import, 0, len(used))
	for _, import_ := range imports {
		if _, ok := used[import_.Name]; ok {
			selected = append(selected, import_)
		}
	}
	return selected
}
