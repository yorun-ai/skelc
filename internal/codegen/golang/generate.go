package golang

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	gomodule "go.yorun.ai/skelc/internal/codegen/golang/module"
	"go.yorun.ai/skelc/internal/codegen/golang/schema"
	"go.yorun.ai/skelc/internal/codegen/golang/source"
	"go.yorun.ai/skelc/internal/codegen/golang/view"
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

func Generate(domain *model.Domain, option Option) error {
	if err := common.ValidateDomain(domain); err != nil {
		return fmt.Errorf("validate Go generation model: %w", err)
	}
	if option.PubOut == "" {
		gen, err := newGen(_GenOption{
			CompilerVersion: option.CompilerVersion,
			ModulePrefix:    option.ModulePrefix,
			Module:          option.Module,
			VineVersion:     option.VineVersion,
			Imports:         option.Imports,
			Mode:            view.ModeFull,
			Domain:          domain,
			Out:             option.Out,
			AsModule:        option.AsModule,
		})
		if err != nil {
			return err
		}
		return gen.gen()
	}

	pubModule := option.PubModule
	if pubModule == "" {
		pubModule = DefaultPubModuleName(option.ModulePrefix, option.Module, domain.Name())
	}
	pubGen, err := newGen(_GenOption{
		CompilerVersion: option.CompilerVersion,
		ModulePrefix:    option.ModulePrefix,
		Module:          pubModule,
		VineVersion:     option.VineVersion,
		Imports:         option.Imports,
		Mode:            view.ModePub,
		Domain:          domain,
		Out:             option.PubOut,
		AsModule:        true,
	})
	if err != nil {
		return err
	}
	if err := pubGen.gen(); err != nil {
		return err
	}
	regularGen, err := newGen(_GenOption{
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
	})
	if err != nil {
		return err
	}
	return regularGen.gen()
}

func newGen(option _GenOption) (*_Gen, error) {
	g := &_Gen{
		domain:            option.Domain,
		mode:              option.Mode,
		asModule:          option.AsModule,
		compilerVersion:   option.CompilerVersion,
		modulePrefix:      option.ModulePrefix,
		goImports:         option.Imports,
		pubImportPath:     option.PubImportPath,
		extraDependencies: option.ExtraDependencies,
		out:               option.Out,
	}
	var err error
	g.vineVersion, err = gomodule.ResolveVineVersion(option.VineVersion)
	if err != nil {
		return nil, err
	}
	g.view, err = view.Build(option.Mode, option.Domain)
	if err != nil {
		return nil, err
	}
	domainParts := strings.Split(g.domain.Name(), ".")
	if option.AsModule {
		g.modName = option.Module
		if g.modName == "" {
			g.modName = buildModuleName(g.modulePrefix, domainParts, option.Mode == view.ModePub)
		}
	}
	g.pkgName, err = inferPackageName(option.Out, packageNameFallback(domainParts, option.Mode == view.ModePub), option.AsModule)
	if err != nil {
		return nil, err
	}
	if err := g.resolveExternalTypeImports(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *_Gen) gen() error {
	if g.asModule {
		if err := gomodule.Generate(gomodule.Option{
			Out:               g.out,
			Module:            g.modName,
			VineVersion:       g.vineVersion,
			Imports:           g.goImports,
			ExtraDependencies: g.extraDependencies,
		}); err != nil {
			return err
		}
	}
	if err := source.GenerateValidated(source.Option{
		Domain:        g.domain,
		View:          g.view,
		Mode:          g.mode,
		PackageName:   g.pkgName,
		PubImportPath: g.pubImportPath,
		Out:           g.out,
	}); err != nil {
		return err
	}
	return schema.GenerateValidated(schema.Option{
		Domain:          g.domain,
		View:            g.view,
		Mode:            g.mode,
		PackageName:     g.pkgName,
		CompilerVersion: g.compilerVersion,
		Out:             g.out,
	})
}
