package skeleton

import (
	"fmt"
	"text/template"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/model"
)

type _Gen struct {
	domain *model.Domain

	renderer *common.Renderer
	template *template.Template
}

func Generate(domain *model.Domain, option Option) error {
	if !option.PubOnly {
		return fmt.Errorf("Skel generation requires public-only output")
	}
	gen := newGen(domain, option.Out)
	if err := gen.generate(); err != nil {
		return err
	}
	return gen.renderer.Err()
}

func newGen(domain *model.Domain, outputDir string) *_Gen {
	return &_Gen{
		domain:   domain,
		renderer: common.NewRenderer(outputDir),
		template: newSkelTemplate(),
	}
}

func (g *_Gen) generate() error {
	view, err := common.BuildPublicView(g.domain)
	if err != nil {
		return err
	}
	g.renderer.Write("domain.skel", g.renderSkel("domain.skel.tpl", g.buildDomainPayload(nil)))
	if len(view.Actors) > 0 {
		g.renderer.Write("actor.skel", g.renderSkel("actor.skel.tpl", g.buildActorPayload(view.Actors)))
	}
	if len(view.Enums) > 0 || len(view.Data) > 0 || len(view.Configs) > 0 || len(view.Resources) > 0 {
		g.renderer.Write("types.skel", g.renderSkel("types.skel.tpl", g.buildTypesPayload(view)))
	}
	if len(view.Events) > 0 {
		g.renderer.Write("event.skel", g.renderSkel("event.skel.tpl", g.buildEventPayload(view.Events)))
	}
	if len(view.Services) > 0 {
		g.renderer.Write("service.skel", g.renderSkel("service.skel.tpl", g.buildServicePayload(view.Services)))
	}
	return nil
}
