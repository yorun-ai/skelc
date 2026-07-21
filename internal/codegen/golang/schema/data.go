package schema

import "go.yorun.ai/skelc/model"

func (g *_Gen) buildEnumSchema(enum *model.Enum) *_EnumSchema {
	schema := &_EnumSchema{
		Name: enum.Name, SkelName: enum.SkelName, Hash: enum.Hash,
		Description: enum.Description,
		Items:       make([]*_EnumItemSchema, 0, len(enum.Items)),
	}
	for _, item := range enum.Items {
		schema.Items = append(schema.Items, &_EnumItemSchema{Name: item.Name, Description: item.Description})
	}
	return schema
}

func (g *_Gen) buildDataSchema(data *model.Data) *_DataSchema {
	schema := &_DataSchema{
		Name: data.Name, SkelName: data.SkelName, Hash: data.Hash,
		Description:    data.Description,
		TypeParameters: make([]string, 0, len(data.TypeParameters)),
		Members:        g.buildMemberSchemas(data.Members),
	}
	for _, typeParameter := range data.TypeParameters {
		schema.TypeParameters = append(schema.TypeParameters, typeParameter.Name)
	}
	return schema
}

func (g *_Gen) buildConfigSchema(config *model.Data) *_ConfigSchema {
	return &_ConfigSchema{
		Name: config.Name, SkelName: config.SkelName, Hash: config.Hash,
		Description: config.Description, Pub: config.Pub,
		Lifecycle: configSchemaLifecycle(config.Lifecycle),
		Members:   g.buildMemberSchemas(config.Members),
	}
}

func configSchemaLifecycle(lifecycle model.ConfigLifecycle) string {
	switch lifecycle {
	case model.ConfigLifecycleEternal:
		return "ETERNAL"
	case model.ConfigLifecycleInstant:
		return "INSTANT"
	default:
		return string(lifecycle)
	}
}

func (g *_Gen) buildEventSchema(event *model.Data) *_EventSchema {
	return &_EventSchema{
		Name: event.Name, SkelName: event.SkelName, Hash: event.Hash,
		Description: event.Description, Pub: event.Pub, Members: g.buildMemberSchemas(event.Members),
	}
}
