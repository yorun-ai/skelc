package source

import (
	"strings"

	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/util/checkutil"
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

	renderer *codegen.Renderer
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

func Generate(domain *model.Domain, outputDir string, option Option) Result {
	gen := newGen(domain, outputDir, option)
	gen.generate()
	return Result{
		PackageName:     gen.pkgName,
		ResolvedImports: gen.resolvedModuleImports(),
	}
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
		renderer:    codegen.NewRenderer(outputDir),
	}
	if g.pkgName == "" {
		scope := g.moduleScope
		if scope == "" {
			scope = packageScope
		}
		g.pkgName = buildPackageName(scope, g.domain.Name(), g.pubOnly)
	}
	g.resolveExternalTypeImports()
	return g
}

func (g *_Gen) generate() {
	checkutil.CheckNilError(g.err, "typescript generator init failed")
	g.validatePubOnly()
	g.genDataTs()
	g.genSpecTs()
	g.genServiceTs()
	g.genIndex()
}
