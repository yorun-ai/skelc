package analyzer

import (
	"fmt"

	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
	"strings"
)

func (p *Analysis) normalize() {
	refs := &refContext{
		enums:    p.enumsMap,
		dataList: p.dataMap,
		imports:  p.importsMap,
	}
	for _, actor := range p.actorsMap {
		if actor.PermService != nil {
			p.normalizeServiceTypes(actor.PermService, refs)
		}
	}
	for _, resource := range p.resourcesMap {
		if resource.CheckService != nil {
			p.normalizeServiceTypes(resource.CheckService, refs)
		}
	}
	for _, service := range p.servicesMap {
		p.checkActorAudiences(service.Audiences, service.Pos, "service", service.Name)
		p.normalizeServiceTypes(service, refs)
		p.normalizeServiceRequire(service)
	}
	for _, web := range p.websMap {
		p.checkActorAudiences(web.Audiences, web.Pos, "web", web.Name)
	}
	for _, task := range p.tasksMap {
		for _, trigger := range task.Triggers {
			for _, arg := range trigger.Arguments {
				fixTypeRef(arg.Type, refs)
			}
		}
	}
	for _, dataType := range p.dataMap {
		refs.typeParameters = sliceutil.MapToMap(dataType.TypeParameters, func(typeParam *model.TypeParameter) (string, *model.TypeParameter) {
			return typeParam.Name, typeParam
		})
		for _, member := range dataType.Members {
			fixTypeRef(member.Type, refs)
		}
	}

	p.checkHardCycleReferences()
	allData := sortData(p.dataMap)
	p.checkDataDoesNotReferenceConfigs(allData)
	p.checkConfigMemberTypes(allData)
	p.finalize()
}

func (p *Analysis) checkActorAudiences(audiences []*model.ActorAudience, ownerPos fmt.Stringer, ownerKind string, ownerName string) {
	for _, audience := range audiences {
		actor := p.actorByRef(audience.Actor)
		checkutil.CheckNotNil(
			actor,
			`%s %s %s references undefined actor "%s"`,
			ownerPos, ownerKind, ownerName, audience.Actor,
		)
		if audience.Via == "" {
			continue
		}
		_, ok := sliceutil.Find(actor.Vias, func(via *model.ActorVia) bool {
			return via.Name == audience.Via
		})
		checkutil.Check(ok,
			`%s %s %s for %s references undefined actor via "%s"`,
			ownerPos, ownerKind, ownerName, audience.Actor, audience.Via,
		)
	}
}

func (p *Analysis) actorByRef(actorName string) *model.Actor {
	qualifier, name, ok := strings.Cut(actorName, ".")
	if !ok {
		return p.actorsMap[actorName]
	}
	import_ := p.importsMap[qualifier]
	if import_ == nil {
		return nil
	}
	return import_.Domain.actorsMap[name]
}

func (p *Analysis) resourceByRef(resourceName string) *model.Resource {
	qualifier, name, ok := strings.Cut(resourceName, ".")
	if !ok {
		return p.resourcesMap[resourceName]
	}
	import_ := p.importsMap[qualifier]
	if import_ == nil {
		return nil
	}
	return import_.Domain.resourcesMap[name]
}
