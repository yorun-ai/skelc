package schema

import "go.yorun.ai/skelc/internal/codegen/golang/view"

func (g *_Gen) buildDomainSchema() *_DomainSchema {
	domainView := g.view
	if g.isSplitRegular() {
		domainView = view.Full(g.Domain)
	}
	schema := &_DomainSchema{
		Domain:      g.Domain.Name(),
		Description: g.Domain.Description(),
		Hash:        g.Domain.Hash(),
		Full:        !g.isSplitPub(),
		Enums:       make([]*_EnumSchema, 0, len(domainView.Enums)),
		Data:        make([]*_DataSchema, 0, len(domainView.Data)),
		Configs:     make([]*_ConfigSchema, 0, len(domainView.Configs)),
		Webs:        make([]*_WebSchema, 0, len(domainView.Webs)),
		Events:      make([]*_EventSchema, 0, len(domainView.Events)),
		Actors:      make([]*_ActorSchema, 0, len(domainView.Actors)),
		Resources:   make([]*_ResourceSchema, 0, len(domainView.Resources)),
		Services:    make([]*_ServiceSchema, 0, len(domainView.Services)),
		Tasks:       make([]*_TaskSchema, 0, len(domainView.Tasks)),
		Generated:   &_GeneratedInfo{CompilerVersion: g.compilerVersion},
	}
	for _, enum := range domainView.Enums {
		schema.Enums = append(schema.Enums, g.buildEnumSchema(enum))
	}
	for _, data := range domainView.Data {
		schema.Data = append(schema.Data, g.buildDataSchema(data))
	}
	for _, config := range domainView.Configs {
		schema.Configs = append(schema.Configs, g.buildConfigSchema(config))
	}
	for _, web := range domainView.Webs {
		schema.Webs = append(schema.Webs, g.buildWebSchema(web))
	}
	for _, event := range domainView.Events {
		schema.Events = append(schema.Events, g.buildEventSchema(event))
	}
	for _, actor := range domainView.Actors {
		schema.Actors = append(schema.Actors, g.buildActorSchema(actor))
	}
	for _, resource := range domainView.Resources {
		schema.Resources = append(schema.Resources, g.buildResourceSchema(resource))
	}
	for _, service := range domainView.Services {
		schema.Services = append(schema.Services, g.buildServiceSchema(service))
	}
	for _, task := range domainView.Tasks {
		schema.Tasks = append(schema.Tasks, g.buildTaskSchema(task))
	}
	return schema
}
