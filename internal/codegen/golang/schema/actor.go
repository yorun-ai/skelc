package schema

import (
	"strings"

	"go.yorun.ai/skelc/model"
)

func (g *_Gen) buildActorSchema(actor *model.Actor) *_ActorSchema {
	schema := &_ActorSchema{
		Name: actor.Name, SkelName: actor.SkelName, Hash: actor.Hash,
		Description: actor.Description, Vias: make([]_ActorVia, 0, len(actor.Vias)),
		AuthEnabled: actor.AuthEnabled, PermEnabled: actor.PermEnabled,
	}
	for _, via := range actor.Vias {
		schema.Vias = append(schema.Vias, actorVia(via.Name))
	}
	if actor.AuthEnabled {
		schema.AuthCredential = g.buildDataSchema(actor.AuthCredential)
		schema.AuthInfo = g.buildDataSchema(actor.AuthInfo)
		schema.AuthService = g.buildServiceSchema(actor.AuthService)
		schema.AuthMethod = g.buildMethodSchema(actor.AuthMethod)
	}
	if actor.PermService != nil {
		schema.PermService = g.buildServiceSchema(actor.PermService)
	}
	if actor.PermMethod != nil {
		schema.PermMethod = g.buildMethodSchema(actor.PermMethod)
	}
	return schema
}

func (g *_Gen) buildWebSchema(web *model.Web) *_WebSchema {
	return &_WebSchema{
		Name: web.Name, SkelName: web.SkelName, Hash: web.Hash,
		Description: web.Description, Audiences: g.buildActorAudienceSchemas(web.Audiences),
	}
}

func (g *_Gen) buildActorAudienceSchemas(audiences []*model.ActorAudience) []*_ActorAudienceSchema {
	schemas := make([]*_ActorAudienceSchema, 0, len(audiences))
	for _, audience := range audiences {
		name := audience.Actor
		if _, baseName, ok := strings.Cut(audience.Actor, "."); ok {
			name = baseName
		}
		schema := &_ActorAudienceSchema{Name: name, SkelName: g.skelName(audience.Actor)}
		if audience.Via != "" {
			schema.Via = actorVia(audience.Via)
		}
		schemas = append(schemas, schema)
	}
	return schemas
}
