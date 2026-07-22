package analyzer

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/model"
)

func buildResourceCheckService(domainName string, resource *model.Resource) *model.Service {
	methods := []*model.Method{}
	serviceName := resource.Name + "CheckService"
	for _, check := range resource.Checks {
		methods = append(methods, prepareResourceCheckMethod(domainName, serviceName, check))
	}
	for _, action := range resource.Actions {
		for _, check := range action.Checks {
			methods = append(methods, prepareResourceCheckMethod(domainName, serviceName, check))
		}
	}
	if len(methods) == 0 {
		return nil
	}
	return &model.Service{
		Pos:      resource.Pos,
		Name:     serviceName,
		SkelName: domainName + "." + serviceName,
		Auth:     model.AuthModeAuth,
		Methods:  methods,
	}
}

func prepareResourceCheckMethod(domainName string, serviceName string, check *model.ResourceCheck) *model.Method {
	method := check.Method
	if method.ArgumentsData != nil {
		method.ArgumentsData.Name = serviceName + method.ArgumentsData.Name
		method.ArgumentsData.Domain = domainName
		method.ArgumentsData.SkelName = domainName + "." + method.ArgumentsData.Name
	}
	return method
}

func parseResource(ge *grammar.Resource, pub bool) *model.Resource {
	checkCase("Resource", caseTypeCamel, ge.Name)
	meta := parseDecoratorMeta(ge.Decorators, decoratorContext{
		allowDesc: true,
	})
	checkutil.CheckNot(meta.HasExample, "%s resource does not support decorator @example", ge.Name.Pos)

	checks := make([]*model.ResourceCheck, 0, len(ge.Checks))
	checkPos := map[string]lexer.Position{}
	for _, grammarCheck := range ge.Checks {
		check := parseResourceCheck("", grammarCheck)
		duplicatedPosition, duplicated := checkPos[check.Name]
		checkutil.CheckFuncAt(grammarCheck.Name.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated resource check %s found, also present at %s", grammarCheck.Name.Pos, check.Name, duplicatedPosition)
		})
		checkPos[check.Name] = grammarCheck.Name.Pos
		checks = append(checks, check)
	}

	actions := make([]*model.ResourceAction, 0, len(ge.Actions))
	actionPos := map[string]lexer.Position{}
	for _, grammarAction := range ge.Actions {
		action := parseResourceAction(grammarAction, checkPos)
		duplicatedPosition, duplicated := actionPos[action.Name]
		checkutil.CheckFuncAt(action.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated resource action %s found, also present at %s", action.Pos, action.Name, duplicatedPosition)
		})
		actionPos[action.Name] = lexer.Position{Filename: action.Pos.File, Line: action.Pos.Line, Column: action.Pos.Column}
		actions = append(actions, action)
	}
	checkutil.Check(len(actions) > 0, "%s resource %s must have at least one action", ge.Name.Pos, ge.Name.Value)

	return &model.Resource{
		Pos:         position(ge.Name.Pos),
		Name:        ge.Name.Value,
		Description: meta.Description,
		Pub:         pub,
		Checks:      checks,
		Actions:     actions,
	}
}

func parseResourceAction(ga *grammar.ResourceAction, resourceCheckPos map[string]lexer.Position) *model.ResourceAction {
	checkCase("ResourceAction", caseTypeLowerCamel, ga.Name)
	meta := parseDecoratorMeta(ga.Decorators, decoratorContext{
		allowDesc: true,
	})
	checkutil.CheckNot(meta.HasExample, "%s resource action does not support decorator @example", ga.Name.Pos)

	checks := make([]*model.ResourceCheck, 0, len(ga.Checks))
	checkPos := map[string]lexer.Position{}
	for _, grammarCheck := range ga.Checks {
		check := parseResourceCheck(ga.Name.Value, grammarCheck)
		if duplicatedPosition, duplicated := resourceCheckPos[check.Name]; duplicated {
			checkutil.Failf("%s duplicated resource action check %s found, also present at %s", grammarCheck.Name.Pos, check.Name, duplicatedPosition)
		}
		duplicatedPosition, duplicated := checkPos[check.Name]
		checkutil.CheckFuncAt(grammarCheck.Name.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated resource action check %s found, also present at %s", grammarCheck.Name.Pos, check.Name, duplicatedPosition)
		})
		checkPos[check.Name] = grammarCheck.Name.Pos
		checks = append(checks, check)
	}

	return &model.ResourceAction{
		Pos:         position(ga.Name.Pos),
		Name:        ga.Name.Value,
		Description: meta.Description,
		Checks:      checks,
	}
}

func parseResourceCheck(actionName string, gc *grammar.ResourceCheck) *model.ResourceCheck {
	checkCase("ResourceCheck", caseTypeLowerCamel, gc.Name)
	args := make([]*model.Argument, 0, len(gc.Arguments)+1)
	hasPermissionCodeArgument := false
	for index, grammarArgument := range gc.Arguments {
		arg := parseArgument(grammarArgument)
		if isPermissionCodeType(arg.Type) {
			checkutil.Check(index == 0 && arg.Name == "code",
				`%s resource check PermissionCode argument must be the first argument named "code"`, grammarArgument.Name.Pos)
			hasPermissionCodeArgument = true
		} else {
			checkutil.Check(arg.Name != "code", `%s resource check argument name "code" is reserved`, grammarArgument.Name.Pos)
		}
		args = append(args, arg)
	}
	if !hasPermissionCodeArgument {
		args = append([]*model.Argument{newPermissionCodeArgument()}, args...)
	}
	methodName := "check" + nameutil.ToCamel(actionName) + nameutil.ToCamel(gc.Name.Value)
	if actionName == "" {
		methodName = "check" + nameutil.ToCamel(gc.Name.Value)
	}
	method := &model.Method{
		Pos:       position(gc.Name.Pos),
		Name:      methodName,
		SkelName:  methodName,
		Auth:      model.AuthModeAuth,
		Arguments: args,
	}
	if len(args) > 0 {
		method.ArgumentsData = &model.Data{
			Name:    fmt.Sprintf("%sArguments", nameutil.ToCamel(method.Name)),
			Members: buildArgumentMembers(method.Arguments),
		}
	}
	return &model.ResourceCheck{
		Name:   gc.Name.Value,
		Method: method,
	}
}

func newPermissionCodeArgument() *model.Argument {
	return &model.Argument{
		Name: "code",
		Type: &model.Type{Kind: model.TypeKindSkelPermissionCode},
	}
}
