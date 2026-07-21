package golang

import (
	"strings"

	gomodule "go.yorun.ai/skelc/internal/codegen/golang/module"
	"go.yorun.ai/skelc/internal/codegen/golang/schema"
	"go.yorun.ai/skelc/internal/codegen/golang/source"
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

type _Gen struct {
	domain *model.Domain
	view   *view.Domain

	mode              view.Mode
	modName           string
	pkgName           string
	asModule          bool
	compilerVersion   string
	vineVersion       string
	modulePrefix      string
	goImports         map[string]string
	pubImportPath     string
	extraDependencies []string
	out               string
}

func Generate(domain *model.Domain, option Option) {
	if option.PubOut == "" {
		newGen(_GenOption{
			CompilerVersion: option.CompilerVersion,
			ModulePrefix:    option.ModulePrefix,
			Module:          option.Module,
			VineVersion:     option.VineVersion,
			Imports:         option.Imports,
			Mode:            view.ModeFull,
			Domain:          domain,
			Out:             option.Out,
			AsModule:        option.AsModule,
		}).gen()
		return
	}

	pubModule := option.PubModule
	if pubModule == "" {
		pubModule = DefaultPubModuleName(option.ModulePrefix, option.Module, domain.Name())
	}
	newGen(_GenOption{
		CompilerVersion: option.CompilerVersion,
		ModulePrefix:    option.ModulePrefix,
		Module:          pubModule,
		VineVersion:     option.VineVersion,
		Imports:         option.Imports,
		Mode:            view.ModePub,
		Domain:          domain,
		Out:             option.PubOut,
		AsModule:        true,
	}).gen()
	newGen(_GenOption{
		CompilerVersion:   option.CompilerVersion,
		ModulePrefix:      option.ModulePrefix,
		Module:            option.Module,
		VineVersion:       option.VineVersion,
		Imports:           option.Imports,
		Domain:            domain,
		Out:               option.Out,
		AsModule:          true,
		Mode:              view.ModeRegular,
		PubImportPath:     pubModule,
		ExtraDependencies: []string{pubModule},
	}).gen()
}

func newGen(option _GenOption) *_Gen {
	checkutil.Check(isValidMode(option.Mode), "invalid go generation mode %q", option.Mode)
	g := &_Gen{
		domain:            option.Domain,
		mode:              option.Mode,
		asModule:          option.AsModule,
		compilerVersion:   option.CompilerVersion,
		vineVersion:       gomodule.ResolveVineVersion(option.VineVersion),
		modulePrefix:      option.ModulePrefix,
		goImports:         option.Imports,
		pubImportPath:     option.PubImportPath,
		extraDependencies: option.ExtraDependencies,
		out:               option.Out,
	}
	g.view = view.New(option.Mode, option.Domain)
	domainParts := strings.Split(g.domain.Name(), ".")
	if option.AsModule {
		g.modName = option.Module
		if g.modName == "" {
			g.modName = buildModuleName(g.modulePrefix, domainParts, option.Mode == view.ModePub)
		}
	}
	g.pkgName = inferPackageName(option.Out, packageNameFallback(domainParts, option.Mode == view.ModePub), option.AsModule)
	g.resolveExternalTypeImports()
	return g
}

func (g *_Gen) gen() {
	if g.asModule {
		gomodule.Generate(gomodule.Option{
			Out:               g.out,
			Module:            g.modName,
			VineVersion:       g.vineVersion,
			Imports:           g.goImports,
			ExtraDependencies: g.extraDependencies,
		})
	}
	source.Generate(source.Option{
		Domain:        g.domain,
		View:          g.view,
		Mode:          g.mode,
		PackageName:   g.pkgName,
		PubImportPath: g.pubImportPath,
		Out:           g.out,
	})
	schema.Generate(schema.Option{
		Domain:          g.domain,
		View:            g.view,
		Mode:            g.mode,
		PackageName:     g.pkgName,
		CompilerVersion: g.compilerVersion,
		Out:             g.out,
	})
}
