package golang

import (
	"fmt"
	"strings"

	gomodule "go.yorun.ai/skelc/internal/codegen/golang/module"
	"go.yorun.ai/skelc/model"
)

func (g *_Gen) resolveExternalTypeImports() error {
	var resolveErr error
	g.visitDomainTypes(func(type_ *model.Type) {
		if resolveErr != nil || type_ == nil || type_.ExternalDomain == "" {
			return
		}
		type_.ExternalImportPath, resolveErr = g.goImportPath(type_.ExternalDomain)
		if !type_.ExternalAliasExplicit {
			type_.ExternalAlias = importPackageName(type_.ExternalDomain, true)
		}
	})
	return resolveErr
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

func (g *_Gen) visitDomainTypes(visit func(*model.Type)) {
	for _, dataType := range g.domain.Data() {
		visitDataTypes(dataType, visit)
	}
	for _, config := range g.domain.Configs() {
		visitDataTypes(config, visit)
	}
	for _, event := range g.domain.Events() {
		visitDataTypes(event, visit)
	}
	for _, service := range g.domain.Services() {
		visitServiceTypes(service, visit)
	}
	for _, actor := range g.domain.Actors() {
		visitServiceTypes(actor.AuthService, visit)
		visitServiceTypes(actor.PermService, visit)
	}
	for _, resource := range g.domain.Resources() {
		visitServiceTypes(resource.CheckService, visit)
	}
	for _, task := range g.domain.Tasks() {
		for _, trigger := range task.Triggers {
			for _, argument := range trigger.Arguments {
				visitType(argument.Type, visit)
			}
		}
	}
}

func visitServiceTypes(service *model.Service, visit func(*model.Type)) {
	if service == nil {
		return
	}
	for _, method := range service.Methods {
		visitType(method.ResultType, visit)
		for _, argument := range method.Arguments {
			visitType(argument.Type, visit)
		}
	}
}

func visitDataTypes(dataType *model.Data, visit func(*model.Type)) {
	for _, member := range dataType.Members {
		visitType(member.Type, visit)
	}
}

func visitType(type_ *model.Type, visit func(*model.Type)) {
	if type_ == nil {
		return
	}
	visit(type_)
	switch type_.Kind {
	case model.TypeKindList:
		visitType(type_.List.Value, visit)
	case model.TypeKindMap:
		visitType(type_.Map.Key, visit)
		visitType(type_.Map.Value, visit)
	case model.TypeKindData:
		for _, typeArgument := range type_.TypeArguments {
			visitType(typeArgument, visit)
		}
	}
}
