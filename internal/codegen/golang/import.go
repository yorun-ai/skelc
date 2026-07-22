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
	for _, dataType := range g.domain.Data() {
		if err := visitDataTypes(dataType, visit); err != nil {
			return err
		}
	}
	for _, config := range g.domain.Configs() {
		if err := visitDataTypes(config, visit); err != nil {
			return err
		}
	}
	for _, event := range g.domain.Events() {
		if err := visitDataTypes(event, visit); err != nil {
			return err
		}
	}
	for _, service := range g.domain.Services() {
		if err := visitServiceTypes(service, visit); err != nil {
			return err
		}
	}
	for _, actor := range g.domain.Actors() {
		if err := visitServiceTypes(actor.AuthService, visit); err != nil {
			return err
		}
		if err := visitServiceTypes(actor.PermService, visit); err != nil {
			return err
		}
	}
	for _, resource := range g.domain.Resources() {
		if err := visitServiceTypes(resource.CheckService, visit); err != nil {
			return err
		}
	}
	for _, task := range g.domain.Tasks() {
		for _, trigger := range task.Triggers {
			for _, argument := range trigger.Arguments {
				if err := common.WalkType(argument.Type, visit); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func visitServiceTypes(service *model.Service, visit common.TypeVisitor) error {
	if service == nil {
		return nil
	}
	for _, method := range service.Methods {
		if err := common.WalkType(method.ResultType, visit); err != nil {
			return err
		}
		for _, argument := range method.Arguments {
			if err := common.WalkType(argument.Type, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

func visitDataTypes(dataType *model.Data, visit common.TypeVisitor) error {
	for _, member := range dataType.Members {
		if err := common.WalkType(member.Type, visit); err != nil {
			return err
		}
	}
	return nil
}
