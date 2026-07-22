package view

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/model"
)

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

func Build(mode Mode, domain *model.Domain) (*Domain, error) {
	if mode == ModePub {
		public, err := common.BuildPublicView(domain)
		if err != nil {
			return nil, err
		}
		return &Domain{
			Enums:     public.Enums,
			Data:      public.Data,
			Configs:   public.Configs,
			Actors:    public.Actors,
			Resources: public.Resources,
			Webs:      []*model.Web{},
			Events:    public.Events,
			Services:  public.Services,
			Tasks:     []*model.Task{},
		}, nil
	}
	if mode == ModeFull {
		return Full(domain), nil
	}
	if mode != ModeRegular {
		return nil, fmt.Errorf("invalid Go generation mode %q", mode)
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
	}, nil
}

// New constructs a generation view and reports invalid modes or public views.
func New(mode Mode, domain *model.Domain) (*Domain, error) { return Build(mode, domain) }
