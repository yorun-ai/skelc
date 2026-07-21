package schema

import "go.yorun.ai/skelc/model"

func (g *_Gen) buildResourceSchema(resource *model.Resource) *_ResourceSchema {
	schema := &_ResourceSchema{
		Name: resource.Name, SkelName: resource.SkelName, Hash: resource.Hash,
		Description: resource.Description, Checks: g.buildResourceCheckSchemas(resource.Checks),
		Actions: make([]*_ResourceActionSchema, 0, len(resource.Actions)),
	}
	for _, action := range resource.Actions {
		schema.Actions = append(schema.Actions, &_ResourceActionSchema{
			Name: action.Name, PermissionCode: action.PermissionCode,
			Description: action.Description, Checks: g.buildResourceCheckSchemas(action.Checks),
		})
	}
	if resource.CheckService != nil {
		schema.CheckService = g.buildServiceSchema(resource.CheckService)
	}
	return schema
}

func (g *_Gen) buildResourceCheckSchemas(checks []*model.ResourceCheck) []*_ResourceCheckSchema {
	schemas := make([]*_ResourceCheckSchema, 0, len(checks))
	for _, check := range checks {
		schemas = append(schemas, &_ResourceCheckSchema{
			Name: check.Name, Method: g.buildMethodSchema(check.Method),
			Arguments: g.buildArgumentSchemas(check.Method.Arguments),
		})
	}
	return schemas
}
