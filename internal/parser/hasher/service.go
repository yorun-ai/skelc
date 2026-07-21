package hasher

import "go.yorun.ai/skelc/model"

func (s *_hashState) resourceHash(resource *model.Resource) string {
	return s.memoHash("resource", resource.SkelName, func() string {
		var checkServiceName string
		var checkServiceHash string
		if resource.CheckService != nil {
			checkServiceName = resource.CheckService.SkelName
			checkServiceHash = s.serviceHash(resource.CheckService)
		}
		return hashValue(_ResourceHashValue{
			Name:             resource.Name,
			SkelName:         resource.SkelName,
			Description:      resource.Description,
			Pub:              resource.Pub,
			Checks:           s.buildResourceCheckHashValues(resource.Checks),
			Actions:          s.buildResourceActionHashValues(resource.Actions),
			CheckService:     checkServiceName,
			CheckServiceHash: checkServiceHash,
		})
	})
}

func (s *_hashState) methodHash(method *model.Method) string {
	return hashValue(_MethodHashValue{
		Name:              method.Name,
		SkelName:          method.SkelName,
		Description:       method.Description,
		Example:           method.Example,
		Auth:              authModeHashValue(method.Auth),
		Require:           s.buildRequireHashValue(method.Require),
		InputDescription:  method.InputDescription,
		OutputDescription: method.OutputDescription,
		OutputExample:     method.OutputExample,
		Arguments:         s.buildArgumentHashValues(method.Arguments),
		ResultType:        s.buildTypeHashValue(method.ResultType),
	})
}

func (s *_hashState) serviceHash(service *model.Service) string {
	return s.memoHash("service", service.SkelName, func() string {
		for _, method := range service.Methods {
			method.Hash = s.methodHash(method)
		}
		return hashValue(_ServiceHashValue{
			Name:        service.Name,
			SkelName:    service.SkelName,
			Description: service.Description,
			Pub:         service.Pub,
			Actors:      s.buildActorAudienceHashValues(service.Audiences),
			Auth:        authModeHashValue(service.Auth),
			Require:     s.buildRequireHashValue(service.Require),
			Methods: buildNamedValues(service.Methods,
				func(method *model.Method) string { return method.SkelName },
				func(method *model.Method) string { return method.Hash }),
		})
	})
}

func (s *_hashState) buildResourceCheckHashValues(checks []*model.ResourceCheck) []*_ResourceCheck {
	values := make([]*_ResourceCheck, 0, len(checks))
	for _, check := range checks {
		values = append(values, &_ResourceCheck{
			Name:       check.Name,
			Method:     check.Method.SkelName,
			MethodHash: s.methodHash(check.Method),
			Arguments:  s.buildArgumentHashValues(check.Method.Arguments),
		})
	}
	return values
}

func (s *_hashState) buildResourceActionHashValues(actions []*model.ResourceAction) []*_ResourceAction {
	values := make([]*_ResourceAction, 0, len(actions))
	for _, action := range actions {
		value := &_ResourceAction{
			Name:           action.Name,
			PermissionCode: action.PermissionCode,
			Description:    action.Description,
			Checks:         s.buildResourceCheckHashValues(action.Checks),
		}
		values = append(values, value)
	}
	return values
}

func (s *_hashState) buildRequireHashValue(require *model.PermissionRequire) *_RequireHashValue {
	if require == nil {
		return nil
	}
	return &_RequireHashValue{
		Expr: s.buildRequireExprHashValue(require.Expr),
	}
}

func (s *_hashState) buildRequireExprHashValue(expr *model.PermissionExpr) *_RequireExprHashValue {
	if expr == nil {
		return nil
	}
	value := &_RequireExprHashValue{
		Mode: string(expr.Mode),
		Code: expr.Code,
	}
	if expr.Check != nil {
		value.Check = &_RequireCheckHashValue{
			ResourceSkelName: expr.Check.ResourceSkelName,
			ActionName:       expr.Check.ActionName,
			CheckName:        expr.Check.CheckName,
			Arguments:        s.buildRequireCheckArgumentHashValues(expr.Check.Arguments),
		}
	}
	if len(expr.Children) > 0 {
		value.Children = make([]*_RequireExprHashValue, 0, len(expr.Children))
		for _, child := range expr.Children {
			value.Children = append(value.Children, s.buildRequireExprHashValue(child))
		}
	}
	return value
}

func (s *_hashState) buildRequireCheckArgumentHashValues(arguments []*model.PermissionCheckArgument) []*_RequireCheckArgument {
	values := make([]*_RequireCheckArgument, 0, len(arguments))
	for _, argument := range arguments {
		values = append(values, &_RequireCheckArgument{
			Name:     argument.Name,
			JsonPath: argument.JsonPath,
			Type:     s.buildTypeHashValue(argument.Type),
		})
	}
	return values
}

func authModeHashValue(mode model.AuthMode) string {
	if mode == "" {
		return string(model.AuthModeUnset)
	}
	return string(mode)
}
