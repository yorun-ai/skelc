package schema

import "go.yorun.ai/skelc/model"

func (g *_Gen) buildServiceSchema(service *model.Service) *_ServiceSchema {
	schema := &_ServiceSchema{
		Name:        service.Name,
		SkelName:    service.SkelName,
		Hash:        service.Hash,
		Description: service.Description,
		Pub:         service.Pub,
		AuthMode:    authMode(service.Auth),
		Audiences:   g.buildActorAudienceSchemas(service.Audiences),
		Require:     g.buildPermRequireSchema(service.Require),
		Methods:     make([]*_MethodSchema, 0, len(service.Methods)),
	}
	for _, method := range service.Methods {
		schema.Methods = append(schema.Methods, g.buildMethodSchema(method))
	}
	return schema
}

func (g *_Gen) buildMethodSchema(method *model.Method) *_MethodSchema {
	return &_MethodSchema{
		Name:              method.Name,
		SkelName:          method.SkelName,
		Hash:              method.Hash,
		Description:       method.Description,
		Example:           method.Example,
		AuthMode:          authMode(method.Auth),
		Require:           g.buildPermRequireSchema(method.Require),
		InputDescription:  method.InputDescription,
		OutputDescription: method.OutputDescription,
		OutputExample:     method.OutputExample,
		Arguments:         g.buildArgumentSchemas(method.Arguments),
		ResultType:        g.buildTypeSchema(method.ResultType),
	}
}

func authMode(mode model.AuthMode) _AuthMode {
	if mode == "" {
		return authModeUnset
	}
	switch mode {
	case model.AuthModeUnset:
		return authModeUnset
	case model.AuthModeAuth:
		return authModeAuth
	case model.AuthModeNoAuth:
		return authModeNoAuth
	default:
		panic("unexpected auth mode " + mode)
	}
}
