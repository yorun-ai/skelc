package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/codegen/typescript/module"
	"go.yorun.ai/skelc/model"
)

func (g *_Gen) resolveExternalTypeImports() {
	visitTypes := g.visitDomainTypes
	if g.pubOnly {
		visitTypes = g.visitPubOnlyTypes
	}
	visitTypes(func(type_ *model.Type) {
		if type_ == nil || type_.ExternalDomain == "" {
			return
		}
		importPath := g.tsImportPath(type_.ExternalDomain)
		type_.ExternalImportPath = importPath
		if !type_.ExternalAliasExplicit {
			type_.ExternalAlias = importPackageAlias(type_.ExternalDomain, true)
		}
	})
}

func (g *_Gen) tsImportPath(domainName string) string {
	if g.err != nil {
		return ""
	}
	if path := g.tsImports[domainName]; path != "" {
		var importPath string
		importPath, g.err = module.ImportPath(path)
		return importPath
	}
	if g.moduleScope == "" {
		g.err = fmt.Errorf("missing TypeScript import for domain %s; pass --ts-import %s=PACKAGE or --ts-module-scope", domainName, domainName)
		return ""
	}
	return buildPackageName(g.moduleScope, domainName, true)
}

func (g *_Gen) resolvedModuleImports() map[string]string {
	imports := make(map[string]string, len(g.domain.Imports()))
	for _, import_ := range g.domain.Imports() {
		imports[import_.Domain.Name()] = g.tsImportPath(import_.Domain.Name())
	}
	return imports
}

func (g *_Gen) visitDomainTypes(visit func(*model.Type)) {
	types := make([]*model.Type, 0)
	for _, dataType := range g.domain.Data() {
		types = appendDataTypeRoots(types, dataType)
	}
	for _, config := range g.domain.Configs() {
		types = appendDataTypeRoots(types, config)
	}
	for _, event := range g.domain.Events() {
		types = appendDataTypeRoots(types, event)
	}
	for _, service := range g.domain.Services() {
		for _, method := range service.Methods {
			types = append(types, method.ResultType)
			for _, arg := range method.Arguments {
				types = append(types, arg.Type)
			}
		}
	}
	for _, task := range g.domain.Tasks() {
		for _, trigger := range task.Triggers {
			for _, arg := range trigger.Arguments {
				types = append(types, arg.Type)
			}
		}
	}
	common.VisitTypes(types, visit)
}

func (g *_Gen) visitPubOnlyTypes(visit func(*model.Type)) {
	types := make([]*model.Type, 0)
	for _, dataType := range g.publicView.Data {
		types = appendDataTypeRoots(types, dataType)
	}
	for _, service := range g.clientServices() {
		if !service.Pub {
			continue
		}
		for _, method := range service.Methods {
			types = append(types, method.ResultType)
			for _, arg := range method.Arguments {
				types = append(types, arg.Type)
			}
		}
	}
	common.VisitTypes(types, visit)
}

func appendDataTypeRoots(types []*model.Type, dataType *model.Data) []*model.Type {
	for _, member := range dataType.Members {
		types = append(types, member.Type)
	}
	return types
}
