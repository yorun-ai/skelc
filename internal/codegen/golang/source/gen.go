package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/model"
)

type _Gen struct {
	Domain *model.Domain

	mode          view.Mode
	pkgName       string
	pubImportPath string
	view          *view.Domain

	Renderer *common.Renderer
}

type Option struct {
	Domain        *model.Domain
	View          *view.Domain
	Mode          view.Mode
	PackageName   string
	PubImportPath string
	Out           string
}

func Generate(option Option) error {
	if err := common.ValidateDomain(option.Domain); err != nil {
		return fmt.Errorf("validate Go source model: %w", err)
	}
	gen := newGen(option)
	gen.gen()
	return gen.Renderer.Err()
}

func newGen(option Option) *_Gen {
	return &_Gen{
		Domain:        option.Domain,
		mode:          option.Mode,
		pkgName:       option.PackageName,
		pubImportPath: option.PubImportPath,
		view:          option.View,
		Renderer:      common.NewRenderer(option.Out),
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
