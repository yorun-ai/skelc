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

	reporter    *diagnosticReporter
	invalidData map[*model.Data]bool
	unavailable map[string]bool
}

type domainImport struct {
	Domain *Analysis
	Model  *model.Import
}

// Analyze analyzes a domain and explicitly reports independent
// validation failures. Invalid declarations are excluded from later global
// validation stages to avoid dependent cascade diagnostics.
func Analyze(content *grammar.SkelContent, importedDomains []*Analysis) (*Analysis, []error) {
	domain := newAnalysis(content)
	domainByName := map[string]*Analysis{}
	for _, importedDomain := range importedDomains {
		domainByName[importedDomain.Model().Name()] = importedDomain
	}
	if !domain.load() {
		return domain, domain.reporter.result()
	}
	diagnosticsBeforeImports := len(domain.reporter.errors)
	domain.loadImports(domainByName)
	if len(domain.reporter.errors) == diagnosticsBeforeImports {
		domain.normalize()
	}
	if len(domain.reporter.errors) == 0 {
		domain.finalize()
	}
	return domain, domain.reporter.result()
}

func AnalyzeImport(content *grammar.SkelContent) (*Analysis, []error) {
	domain := newAnalysis(content)
	if domain.load() {
		domain.normalizeImport()
	}
	if len(domain.reporter.errors) == 0 {
		domain.finalize()
	}
	return domain, domain.reporter.result()
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

		reporter:    newDiagnosticReporter(),
		invalidData: map[*model.Data]bool{},
		unavailable: map[string]bool{},
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
