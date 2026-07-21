package analyzer

import (
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

type Analysis struct {
	name        string
	description string
	model       *model.Domain

	imports []*model.Import

	enums     []*model.Enum
	dataList  []*model.Data
	configs   []*model.Data
	events    []*model.Data
	actors    []*model.Actor
	resources []*model.Resource
	webs      []*model.Web
	services  []*model.Service
	tasks     []*model.Task

	warnings []string

	content *grammar.SkelContent

	enumsMap     map[string]*model.Enum
	dataMap      map[string]*model.Data
	actorsMap    map[string]*model.Actor
	resourcesMap map[string]*model.Resource
	websMap      map[string]*model.Web
	servicesMap  map[string]*model.Service
	tasksMap     map[string]*model.Task
	importsMap   map[string]*domainImport
}

type domainImport struct {
	Domain *Analysis
	Model  *model.Import
}

func Analyze(content *grammar.SkelContent) *Analysis {
	return AnalyzeWithImports(content, nil)
}

func AnalyzeWithImports(content *grammar.SkelContent, importedDomains []*Analysis) *Analysis {
	domain := newAnalysis(content)
	domainByName := map[string]*Analysis{}
	for _, importedDomain := range importedDomains {
		domainByName[importedDomain.Model().Name()] = importedDomain
	}
	domain.load()
	domain.loadImports(domainByName)
	domain.normalize()
	return domain
}

func AnalyzeImport(content *grammar.SkelContent) *Analysis {
	domain := newAnalysis(content)
	domain.load()
	domain.finalize()
	return domain
}

func newAnalysis(content *grammar.SkelContent) *Analysis {
	return &Analysis{
		name: "",

		content: content,

		enumsMap:     map[string]*model.Enum{},
		dataMap:      map[string]*model.Data{},
		actorsMap:    map[string]*model.Actor{},
		resourcesMap: map[string]*model.Resource{},
		websMap:      map[string]*model.Web{},
		servicesMap:  map[string]*model.Service{},
		tasksMap:     map[string]*model.Task{},
		importsMap:   map[string]*domainImport{},
	}
}

func (p *Analysis) Model() *model.Domain {
	if p.model != nil {
		return p.model
	}
	p.model = model.NewDomainFromSpec(model.DomainSpec{
		Name: p.name, Description: p.description,
		Imports: p.imports, Enums: p.enums, Data: p.dataList, Configs: p.configs, Events: p.events,
		Actors: p.actors, Resources: p.resources, Webs: p.webs, Services: p.services, Tasks: p.tasks,
	})
	return p.model
}

func (p *Analysis) Warnings() []string {
	return append([]string{}, p.warnings...)
}
