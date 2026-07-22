package source

import (
	"fmt"

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
		for _, method := range service.Methods {
			visitType(method.ResultType, visit)
			for _, arg := range method.Arguments {
				visitType(arg.Type, visit)
			}
		}
	}
	for _, task := range g.domain.Tasks() {
		for _, trigger := range task.Triggers {
			for _, arg := range trigger.Arguments {
				visitType(arg.Type, visit)
			}
		}
	}
}

func (g *_Gen) visitPubOnlyTypes(visit func(*model.Type)) {
	for _, dataType := range g.publicView.Data {
		visitDataTypes(dataType, visit)
	}
	for _, service := range g.clientServices() {
		if !service.Pub {
			continue
		}
		for _, method := range service.Methods {
			visitType(method.ResultType, visit)
			for _, arg := range method.Arguments {
				visitType(arg.Type, visit)
			}
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
		for _, typeArg := range type_.TypeArguments {
			visitType(typeArg, visit)
		}
	}
}
