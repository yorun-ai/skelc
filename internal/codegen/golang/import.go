package golang

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	gomodule "go.yorun.ai/skelc/internal/codegen/golang/module"
	"go.yorun.ai/skelc/model"
)

func (g *_Gen) resolveExternalTypeImports() error {
	return g.visitDomainTypes(func(type_ *model.Type) error {
		if type_.ExternalDomain == "" {
			return nil
		}
		path, err := g.goImportPath(type_.ExternalDomain)
		if err != nil {
			return err
		}
		type_.ExternalImportPath = path
		if !type_.ExternalAliasExplicit {
			type_.ExternalAlias = importPackageName(type_.ExternalDomain, true)
		}
		return nil
	})
}

func (g *_Gen) goImportPath(domainName string) (string, error) {
	if path := g.goImports[domainName]; path != "" {
		return gomodule.ImportPath(path)
	}
	if g.modulePrefix == "" {
		return "", fmt.Errorf("missing Go import for domain %s; pass --go-import %s=PACKAGE or --go-module-prefix", domainName, domainName)
	}
	return buildModuleName(g.modulePrefix, strings.Split(domainName, "."), true), nil
}

func (g *_Gen) visitDomainTypes(visit common.TypeVisitor) error {
	types := make([]*model.Type, 0)
	for _, dataType := range g.domain.Data() {
		types = appendDataTypes(types, dataType)
	}
	for _, config := range g.domain.Configs() {
		types = appendDataTypes(types, config)
	}
	for _, event := range g.domain.Events() {
		types = appendDataTypes(types, event)
	}
	for _, service := range g.domain.Services() {
		types = appendServiceTypes(types, service)
	}
	for _, actor := range g.domain.Actors() {
		types = appendServiceTypes(types, actor.AuthService)
		types = appendServiceTypes(types, actor.PermService)
	}
	for _, resource := range g.domain.Resources() {
		types = appendServiceTypes(types, resource.CheckService)
	}
	for _, task := range g.domain.Tasks() {
		for _, trigger := range task.Triggers {
			for _, argument := range trigger.Arguments {
				types = append(types, argument.Type)
			}
		}
	}
	return common.WalkTypes(types, visit)
}

func appendServiceTypes(types []*model.Type, service *model.Service) []*model.Type {
	if service == nil {
		return types
	}
	for _, method := range service.Methods {
		types = append(types, method.ResultType)
		for _, argument := range method.Arguments {
			types = append(types, argument.Type)
		}
	}
	return types
}

func appendDataTypes(types []*model.Type, dataType *model.Data) []*model.Type {
	for _, member := range dataType.Members {
		types = append(types, member.Type)
	}
	return types
}
