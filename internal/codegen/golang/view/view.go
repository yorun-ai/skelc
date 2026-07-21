package view

import "go.yorun.ai/skelc/model"

type Mode string

const (
	ModeFull    Mode = "full"
	ModePub     Mode = "pub"
	ModeRegular Mode = "regular"
)

type Domain struct {
	Enums     []*model.Enum
	Data      []*model.Data
	Configs   []*model.Data
	Actors    []*model.Actor
	Resources []*model.Resource
	Webs      []*model.Web
	Events    []*model.Data
	Services  []*model.Service
	Tasks     []*model.Task
}

func Full(domain *model.Domain) *Domain {
	return &Domain{
		Enums:     domain.Enums(),
		Data:      domain.Data(),
		Configs:   domain.Configs(),
		Actors:    domain.Actors(),
		Resources: domain.Resources(),
		Webs:      domain.Webs(),
		Events:    domain.Events(),
		Services:  domain.Services(),
		Tasks:     domain.Tasks(),
	}
}

func newPub(domain *model.Domain) *Domain {
	services := filterPubServices(domain.Services())
	events := filterPubEvents(domain.Events())
	configs := make([]*model.Data, 0)
	for _, config := range domain.Configs() {
		if config.Pub {
			configs = append(configs, config)
		}
	}
	resources := filterPubResources(domain.Resources())
	validatePubView(domain, services, events, configs, resources)

	return &Domain{
		Enums:     filterPubEnums(domain.Enums()),
		Data:      filterPubData(domain.Data()),
		Configs:   configs,
		Actors:    filterPubActors(domain.Actors()),
		Resources: resources,
		Webs:      []*model.Web{},
		Events:    events,
		Services:  services,
		Tasks:     []*model.Task{},
	}
}

func New(mode Mode, domain *model.Domain) *Domain {
	if mode == ModePub {
		return newPub(domain)
	}
	if mode == ModeFull {
		return Full(domain)
	}
	return &Domain{
		Enums:     filterNonPubEnums(domain.Enums()),
		Data:      filterNonPubData(domain.Data()),
		Configs:   filterNonPubData(domain.Configs()),
		Actors:    filterNonPubActors(domain.Actors()),
		Resources: filterNonPubResources(domain.Resources()),
		Webs:      domain.Webs(),
		Events:    domain.Events(),
		Services:  domain.Services(),
		Tasks:     domain.Tasks(),
	}
}
