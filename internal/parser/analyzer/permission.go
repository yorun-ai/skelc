package analyzer

import (
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
	"strings"
)

func parseRequire(gr *grammar.Require) *model.PermissionRequire {
	if gr == nil {
		return nil
	}
	return &model.PermissionRequire{
		Expr: parseRequireExpr(gr.Expr),
	}
}

func parseRequireExpr(expr *grammar.RequireExpr) *model.PermissionExpr {
	if expr.Term != nil {
		return &model.PermissionExpr{Check: parseRequireTerm(expr.Term)}
	}

	mode := model.PermissionRequireMode(expr.Mode)
	checkutil.Check(mode == model.PermissionRequireModeAll || mode == model.PermissionRequireModeAny,
		"unsupported require mode %s", expr.Mode)
	children := make([]*model.PermissionExpr, 0, len(expr.Children))
	for _, child := range expr.Children {
		children = append(children, parseRequireExpr(child))
	}
	return &model.PermissionExpr{
		Mode:     mode,
		Children: children,
	}
}

func parseRequireTerm(term *grammar.RequireTerm) *model.PermissionCheckInvocation {
	resourceRef := term.Target.Resource.String()
	action := term.Target.Action.Value
	item := &model.PermissionCheckInvocation{
		ResourceSkelName: resourceRef,
		ActionName:       action,
	}
	if term.Call == nil {
		return item
	}
	item.CheckName = term.Call.Name.Value
	item.Arguments = make([]*model.PermissionCheckArgument, 0, len(term.Call.Arguments))
	for _, arg := range term.Call.Arguments {
		item.Arguments = append(item.Arguments, &model.PermissionCheckArgument{
			JsonPath: arg.String(),
		})
	}
	return item
}

func (p *Analysis) normalizeServiceTypes(service *model.Service, refs *refContext) {
	for _, method := range service.Methods {
		fixTypeRef(method.ResultType, refs)
		for _, arg := range method.Arguments {
			fixTypeRef(arg.Type, refs)
		}
	}
}

func (p *Analysis) normalizePermissionCodes(codes []string) {
	for index, code := range codes {
		resourceRef, actionName := splitPermissionCode(code)
		resource, action := p.findResourceAction(resourceRef, actionName)
		checkutil.CheckNotNil(resource, `permission %s references undefined resource %s`, code, resourceRef)
		checkutil.CheckNotNil(action, `permission %s references undefined action %s`, code, actionName)
		codes[index] = action.PermissionCode
	}
}

func splitPermissionCode(code string) (string, string) {
	index := strings.LastIndex(code, ":")
	checkutil.Check(index > 0 && index < len(code)-1, `permission %s must be resource:action`, code)
	return code[:index], code[index+1:]
}

func (p *Analysis) normalizeServiceRequire(service *model.Service) {
	p.normalizeRequire(service.Require, false, nil, service.Pos)
	for _, method := range service.Methods {
		p.normalizeRequire(method.Require, true, method, method.Pos)
	}
}

func (p *Analysis) normalizeRequire(require *model.PermissionRequire, allowChecks bool, method *model.Method, ownerPos model.Position) {
	if require == nil {
		return
	}
	require.Expr = p.normalizeRequireExpr(require.Expr, allowChecks, method, ownerPos)
}

func (p *Analysis) normalizeRequireExpr(expr *model.PermissionExpr, allowChecks bool, method *model.Method, ownerPos model.Position) *model.PermissionExpr {
	if expr.Mode == "" && expr.Check != nil {
		return p.normalizeRequireItem(expr.Check, allowChecks, method, ownerPos)
	}

	checkutil.Check(len(expr.Children) > 0, "%s require %s must have at least one item", ownerPos, expr.Mode)
	children := make([]*model.PermissionExpr, 0, len(expr.Children))
	for _, child := range expr.Children {
		children = append(children, p.normalizeRequireExpr(child, allowChecks, method, ownerPos))
	}
	expr.Children = children
	return expr
}

func (p *Analysis) normalizeRequireItem(item *model.PermissionCheckInvocation, allowChecks bool, method *model.Method, ownerPos model.Position) *model.PermissionExpr {
	resourceRef := item.ResourceSkelName
	resource, action := p.findResourceAction(resourceRef, item.ActionName)
	checkutil.CheckNotNil(resource, `%s require references undefined resource "%s"`, ownerPos, resourceRef)
	checkutil.CheckNotNil(action, `%s require references undefined action "%s"`, ownerPos, item.ActionName)
	checkutil.Check(!p.isImportedResourceRef(resourceRef) || resource.Pub,
		`%s require references non-pub resource "%s"`, ownerPos, resourceRef)
	codeExpr := &model.PermissionExpr{
		Mode: model.PermissionRequireModeCode,
		Code: action.PermissionCode,
	}
	if item.CheckName == "" {
		return codeExpr
	}

	checkutil.Check(allowChecks, "%s service require does not support check method", ownerPos)
	check := findResourceCheck(resource, action, item.CheckName)
	checkutil.CheckNotNil(check, `%s require references undefined check "%s"`, ownerPos, item.CheckName)
	checkArguments := resourceCheckArguments(check)
	checkutil.Check(len(item.Arguments) == len(checkArguments),
		"%s require check %s expects %d argument(s), got %d",
		ownerPos, item.CheckName, len(checkArguments), len(item.Arguments))
	for index, argument := range item.Arguments {
		checkArgument := checkArguments[index]
		argument.Name = checkArgument.Name
		argument.Type = checkArgument.Type
		valueType := resolveMethodArgumentJsonPath(method, argument.JsonPath)
		checkutil.Check(typeEqual(valueType, checkArgument.Type),
			"%s require check argument %s expects %s, got %s from %s",
			ownerPos, checkArgument.Name, checkArgument.Type.Name(), valueType.Name(), argument.JsonPath)
	}
	return &model.PermissionExpr{
		Mode: model.PermissionRequireModeAll,
		Children: []*model.PermissionExpr{
			codeExpr,
			{
				Mode: model.PermissionRequireModeCheck,
				Check: &model.PermissionCheckInvocation{
					ResourceSkelName: resource.SkelName,
					ActionName:       item.ActionName,
					CheckName:        item.CheckName,
					ServiceSkelName:  resource.CheckService.SkelName,
					MethodSkelName:   check.Method.SkelName,
					Arguments:        item.Arguments,
				},
			},
		},
	}
}

func resourceCheckArguments(check *model.ResourceCheck) []*model.Argument {
	if len(check.Method.Arguments) > 0 && isPermissionCodeType(check.Method.Arguments[0].Type) {
		return check.Method.Arguments[1:]
	}
	return check.Method.Arguments
}

func isPermissionCodeType(type_ *model.Type) bool {
	return type_ != nil && type_.Kind == model.TypeKindSkelPermissionCode
}

func (p *Analysis) isImportedResourceRef(resourceRef string) bool {
	qualifier, _, ok := strings.Cut(resourceRef, ".")
	return ok && p.importsMap[qualifier] != nil
}

func (p *Analysis) findResourceAction(resourceRef string, actionName string) (*model.Resource, *model.ResourceAction) {
	resource := p.resourceByRef(resourceRef)
	if resource == nil {
		return nil, nil
	}
	for _, action := range resource.Actions {
		if action.Name == actionName {
			return resource, action
		}
	}
	return resource, nil
}

func findResourceCheck(resource *model.Resource, action *model.ResourceAction, checkName string) *model.ResourceCheck {
	for _, check := range resource.Checks {
		if check.Name == checkName {
			return check
		}
	}
	for _, check := range action.Checks {
		if check.Name == checkName {
			return check
		}
	}
	return nil
}
