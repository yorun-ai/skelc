package source

import (
	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/model"
)

type _Gen struct {
	Domain *model.Domain

	mode          view.Mode
	pkgName       string
	pubImportPath string
	view          *view.Domain

	Renderer *codegen.Renderer
}

type Option struct {
	Domain        *model.Domain
	View          *view.Domain
	Mode          view.Mode
	PackageName   string
	PubImportPath string
	Out           string
}

func Generate(option Option) {
	newGen(option).gen()
}

func newGen(option Option) *_Gen {
	return &_Gen{
		Domain:        option.Domain,
		mode:          option.Mode,
		pkgName:       option.PackageName,
		pubImportPath: option.PubImportPath,
		view:          option.View,
		Renderer:      codegen.NewRenderer(option.Out),
	}
}

func (g *_Gen) gen() {
	g.genDocGo()
	g.genEnumGo()
	g.genDataGo()
	g.genConfigGo()
	g.genActorGo()
	g.genWebGo()
	g.genEventGo()
	g.genResourceGo()
	g.genServiceGo()
	g.genTaskGo()
	g.genFacadeGo()
}

func (g *_Gen) isSplitPub() bool {
	return g.mode == view.ModePub
}

func (g *_Gen) isSplitRegular() bool {
	return g.mode == view.ModeRegular
}
