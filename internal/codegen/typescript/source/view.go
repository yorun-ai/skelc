package source

import (
	"fmt"
	"slices"

	"go.yorun.ai/skelc/model"
)

func hasClientActorVia(actor *model.Actor) bool {
	for _, via := range actor.Vias {
		if model.ActorViaKind(via.Name) == model.ActorViaClient {
			return true
		}
	}
	return false
}

func (g *_Gen) clientServices() []*model.Service {
	actorHasClient := make(map[string]bool, len(g.domain.Actors()))
	for _, actor := range g.domain.Actors() {
		actorHasClient[actor.Name] = hasClientActorVia(actor)
	}
	for _, import_ := range g.domain.Imports() {
		for _, actor := range import_.Domain.Actors() {
			actorHasClient[fmt.Sprintf("%s.%s", import_.Alias, actor.Name)] = hasClientActorVia(actor)
		}
	}

	services := make([]*model.Service, 0, len(g.domain.Services()))
	for _, service := range g.domain.Services() {
		for _, audience := range service.Audiences {
			if audience.Via != "" {
				if audience.Via == string(model.ActorViaClient) {
					services = append(services, service)
					break
				}
				continue
			}
			if actorHasClient[audience.Actor] {
				services = append(services, service)
				break
			}
		}
	}
	return services
}

func (g *_Gen) serviceClientServices() []*model.Service {
	services := g.clientServices()
	if g.pubOnly {
		services = slices.DeleteFunc(services, func(service *model.Service) bool {
			return !service.Pub
		})
	}
	return services
}
