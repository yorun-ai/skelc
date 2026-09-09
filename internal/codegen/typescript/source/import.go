package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/codegen/typescript/module"
	"go.yorun.ai/skelc/internal/model"
)

func (g *_Gen) resolveExternalTypeImports() {
	common.VisitTypes(common.ApiTypeRoots(g.apiView.Data, g.apiView.Services), func(type_ *model.Type) {
		if type_ == nil || type_.ExternalDomain == "" {
			return
		}
		importPath := g.tsImportPath(type_.ExternalDomain)
		type_.ExternalImportPath = importPath
		if !type_.ExternalAliasExplicit {
			type_.ExternalAlias = importPackageAlias(type_.ExternalDomain)
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
	return buildPackageName(g.moduleScope, domainName)
}

func (g *_Gen) resolvedModuleImports() map[string]string {
	imports := make(map[string]string, len(g.domain.Imports()))
	common.VisitTypes(common.ApiTypeRoots(g.apiView.Data, g.apiView.Services), func(kind *model.Type) {
		if kind.ExternalDomain != "" {
			imports[kind.ExternalDomain] = g.tsImportPath(kind.ExternalDomain)
		}
	})
	return imports
}
