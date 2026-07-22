package schema

import "go.yorun.ai/skelc/model"

func (g *_Gen) buildPermRequireSchema(require *model.PermissionRequire) *_PermRequire {
	if require == nil {
		return nil
	}
	return &_PermRequire{Expr: g.buildPermExprSchema(require.Expr)}
}

func (g *_Gen) buildPermExprSchema(expr *model.PermissionExpr) *_PermExpr {
	if expr == nil {
		return nil
	}
	schema := &_PermExpr{Mode: permissionRequireMode(expr.Mode), Code: expr.Code}
	if expr.Check != nil {
		schema.Check = &_PermCheckInvocation{
			ResourceSkelName: expr.Check.ResourceSkelName,
			ActionName:       expr.Check.ActionName,
			CheckName:        expr.Check.CheckName,
			ServiceSkelName:  expr.Check.ServiceSkelName,
			MethodSkelName:   expr.Check.MethodSkelName,
			Arguments:        g.buildPermCheckArgumentSchemas(expr.Check.Arguments),
		}
	}
	if len(expr.Children) > 0 {
		schema.Children = make([]*_PermExpr, 0, len(expr.Children))
		for _, child := range expr.Children {
			schema.Children = append(schema.Children, g.buildPermExprSchema(child))
		}
	}
	return schema
}

func (g *_Gen) buildPermCheckArgumentSchemas(arguments []*model.PermissionCheckArgument) []*_PermCheckArgument {
	schemas := make([]*_PermCheckArgument, 0, len(arguments))
	for _, argument := range arguments {
		schemas = append(schemas, &_PermCheckArgument{
			Name: argument.Name, JsonPath: argument.JsonPath, Type: g.buildTypeSchema(argument.Type),
		})
	}
	return schemas
}

func permissionRequireMode(mode model.PermissionRequireMode) _PermRequireMode {
	switch mode {
	case model.PermissionRequireModeCode:
		return permRequireModeCode
	case model.PermissionRequireModeCheck:
		return permRequireModeCheck
	case model.PermissionRequireModeAll:
		return permRequireModeAll
	case model.PermissionRequireModeAny:
		return permRequireModeAny
	default:
		return ""
	}
}
