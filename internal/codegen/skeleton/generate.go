package skeleton

import (
	"text/template"

	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

type _Gen struct {
	domain  *model.Domain
	pubOnly bool

	renderer *codegen.Renderer
	template *template.Template
}

func Generate(domain *model.Domain, option Option) {
	newGen(domain, option.Out, option.PubOnly).generate()
}

func newGen(domain *model.Domain, outputDir string, pubOnly bool) *_Gen {
	return &_Gen{
		domain:   domain,
		pubOnly:  pubOnly,
		renderer: codegen.NewRenderer(outputDir),
		template: newSkelTemplate(),
	}
}

func (g *_Gen) generate() {
	checkutil.Check(g.pubOnly, "gen skel requires pub")
	view := newPubView(g.domain)
	validatePubView(g.domain, view)
	g.renderer.Write("domain.skel", g.renderSkel("domain.skel.tpl", g.buildDomainPayload(nil)))
	if len(view.actors) > 0 {
		g.renderer.Write("actor.skel", g.renderSkel("actor.skel.tpl", g.buildActorPayload(view.actors)))
	}
	if len(view.enums) > 0 || len(view.dataList) > 0 || len(view.configs) > 0 || len(view.resources) > 0 {
		g.renderer.Write("types.skel", g.renderSkel("types.skel.tpl", g.buildTypesPayload(view)))
	}
	if len(view.events) > 0 {
		g.renderer.Write("event.skel", g.renderSkel("event.skel.tpl", g.buildEventPayload(view.events)))
	}
	if len(view.services) > 0 {
		g.renderer.Write("service.skel", g.renderSkel("service.skel.tpl", g.buildServicePayload(view.services)))
	}
}
