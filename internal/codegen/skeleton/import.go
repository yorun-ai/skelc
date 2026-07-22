package skeleton

import (
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/model"
)

func collectTypeImports(domain *model.Domain, view *common.PublicView) []*model.Import {
	used := map[string]struct{}{}
	types := make([]*model.Type, 0)
	for _, data := range view.Data {
		types = appendDataImportTypes(types, data)
	}
	for _, config := range view.Configs {
		types = appendDataImportTypes(types, config)
	}
	for _, resource := range view.Resources {
		for _, check := range resource.Checks {
			for _, argument := range renderResourceCheckArguments(check) {
				types = append(types, argument.Type)
			}
		}
		for _, action := range resource.Actions {
			for _, check := range action.Checks {
				for _, argument := range renderResourceCheckArguments(check) {
					types = append(types, argument.Type)
				}
			}
		}
	}
	collectImportsFromTypes(used, types)
	return selectUsedImports(domain.Imports(), used)
}

func collectDataImports(domain *model.Domain, dataList []*model.Data) []*model.Import {
	used := map[string]struct{}{}
	types := make([]*model.Type, 0)
	for _, data := range dataList {
		types = appendDataImportTypes(types, data)
	}
	collectImportsFromTypes(used, types)
	return selectUsedImports(domain.Imports(), used)
}

func collectActorImports(domain *model.Domain, actors []*model.Actor) []*model.Import {
	used := map[string]struct{}{}
	types := make([]*model.Type, 0)
	for _, actor := range actors {
		if actor.AuthEnabled {
			types = appendDataImportTypes(types, actor.AuthCredential)
			types = appendDataImportTypes(types, actor.AuthInfo)
		}
	}
	collectImportsFromTypes(used, types)
	return selectUsedImports(domain.Imports(), used)
}

func collectServiceImports(domain *model.Domain, services []*model.Service) []*model.Import {
	used := map[string]struct{}{}
	types := make([]*model.Type, 0)
	for _, service := range services {
		for _, audience := range service.Audiences {
			collectImportFromQualifiedName(used, audience.Actor)
		}
		for _, method := range service.Methods {
			types = append(types, method.ResultType)
			for _, argument := range method.Arguments {
				types = append(types, argument.Type)
			}
		}
	}
	collectImportsFromTypes(used, types)
	return selectUsedImports(domain.Imports(), used)
}

func collectImportFromQualifiedName(used map[string]struct{}, name string) {
	qualifier, _, ok := strings.Cut(name, ".")
	if ok {
		used[qualifier] = struct{}{}
	}
}

func appendDataImportTypes(types []*model.Type, data *model.Data) []*model.Type {
	for _, member := range data.Members {
		types = append(types, member.Type)
	}
	return types
}

func collectImportsFromTypes(used map[string]struct{}, types []*model.Type) {
	common.VisitTypes(types, func(current *model.Type) {
		if current.ExternalDomain != "" {
			used[current.ExternalDomain] = struct{}{}
		}
	})
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
