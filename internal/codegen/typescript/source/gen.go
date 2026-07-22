package source

import (
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/model"
)

const packageScope = "@yorun-ai/skeled"

type _Gen struct {
	domain *model.Domain

	pubOnly     bool
	moduleScope string
	pkgName     string
	tsImports   map[string]string
	outputDir   string
	err         error
	publicView  *common.PublicView

	renderer *common.Renderer
}

type Option struct {
	PubOnly     bool
	ModuleScope string
	Module      string
	Imports     map[string]string
}

type Result struct {
	PackageName     string
	ResolvedImports map[string]string
}

// GenerateValidated renders a domain already checked by common.ValidateDomain.
func GenerateValidated(domain *model.Domain, outputDir string, option Option) (Result, error) {
	gen := newGen(domain, outputDir, option)
	if gen.err != nil {
		return Result{}, gen.err
	}
	gen.generate()
	resolvedImports := gen.resolvedModuleImports()
	if gen.err != nil {
		return Result{}, gen.err
	}
	return Result{
		PackageName:     gen.pkgName,
		ResolvedImports: resolvedImports,
	}, gen.renderer.Err()
}

func newGen(domain *model.Domain, outputDir string, options ...Option) *_Gen {
	option := Option{}
	if len(options) > 0 {
		option = options[0]
	}
	g := &_Gen{
		domain:      domain,
		pubOnly:     option.PubOnly,
		moduleScope: strings.TrimRight(option.ModuleScope, "/"),
		pkgName:     strings.TrimRight(option.Module, "/"),
		tsImports:   option.Imports,
		outputDir:   outputDir,
		renderer:    common.NewRenderer(outputDir),
	}
	if g.pkgName == "" {
		scope := g.moduleScope
		if scope == "" {
			scope = packageScope
		}
		g.pkgName = buildPackageName(scope, g.domain.Name(), g.pubOnly)
	}
	if g.pubOnly {
		g.publicView, g.err = common.BuildPublicView(domain)
		if g.err != nil {
			return g
		}
	}
	g.resolveExternalTypeImports()
	return g
}

func (g *_Gen) generate() {
	g.genDataTs()
	g.genSpecTs()
	g.genServiceTs()
	g.genIndex()
}
