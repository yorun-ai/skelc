package common

import (
	"fmt"

	"go.yorun.ai/skelc/model"
)

// ValidateDomain rejects malformed programmatically constructed models before
// target renderers dereference tagged union fields or convert enum values.
func ValidateDomain(domain *model.Domain) error {
	return validateDomain(domain, map[*model.Domain]bool{})
}

func validateDomain(domain *model.Domain, seen map[*model.Domain]bool) error {
	if domain == nil {
		return fmt.Errorf("cannot generate code for a nil domain")
	}
	if seen[domain] {
		return nil
	}
	seen[domain] = true
	for _, domainImport := range domain.Imports() {
		if domainImport == nil {
			return fmt.Errorf("generated model contains nil import")
		}
		if domainImport.Domain == nil {
			return fmt.Errorf("import %s has no domain model", domainImport.Name)
		}
		if err := validateDomain(domainImport.Domain, seen); err != nil {
			return fmt.Errorf("import %s: %w", domainImport.Name, err)
		}
	}
	for _, values := range [][]*model.Data{domain.Data(), domain.Configs(), domain.Events()} {
		for _, data := range values {
			if err := validateData(data); err != nil {
				return err
			}
		}
	}
	for _, enum := range domain.Enums() {
		if enum == nil {
			return fmt.Errorf("generated model contains nil enum")
		}
		if enum.UnspecifiedItem == nil {
			return fmt.Errorf("enum %s has no unspecified item", enum.Name)
		}
		for _, item := range enum.Items {
			if item == nil {
				return fmt.Errorf("enum %s contains a nil item", enum.Name)
			}
		}
	}
	for _, config := range domain.Configs() {
		if config.Lifecycle != model.ConfigLifecycleEternal && config.Lifecycle != model.ConfigLifecycleInstant {
			return fmt.Errorf("config %s has unsupported lifecycle %q", config.Name, config.Lifecycle)
		}
	}
	for _, actor := range domain.Actors() {
		if err := validateActor(actor); err != nil {
			return err
		}
	}
	for _, service := range domain.Services() {
		if err := validateService(service); err != nil {
			return err
		}
	}
	for _, resource := range domain.Resources() {
		if err := validateResource(resource); err != nil {
			return err
		}
	}
	for _, web := range domain.Webs() {
		if web == nil {
			return fmt.Errorf("generated model contains nil web")
		}
		if err := validateAudiences("web "+web.Name, web.Audiences); err != nil {
			return err
		}
	}
	for _, task := range domain.Tasks() {
		if task == nil {
			return fmt.Errorf("generated model contains nil task")
		}
		for _, trigger := range task.Triggers {
			if trigger == nil {
				return fmt.Errorf("task %s contains a nil trigger", task.Name)
			}
			if err := validateArguments("task trigger "+trigger.Name, trigger.Arguments, trigger.ArgumentsData); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateActor(actor *model.Actor) error {
	if actor == nil {
		return fmt.Errorf("generated model contains nil actor")
	}
	for _, via := range actor.Vias {
		if via == nil {
			return fmt.Errorf("actor %s contains a nil via", actor.Name)
		}
		if err := validateActorVia(via.Name); err != nil {
			return fmt.Errorf("actor %s: %w", actor.Name, err)
		}
	}
	if actor.AuthEnabled {
		if actor.AuthCredential == nil || actor.AuthInfo == nil || actor.AuthService == nil || actor.AuthMethod == nil {
			return fmt.Errorf("actor %s has incomplete auth support", actor.Name)
		}
		if err := validateData(actor.AuthCredential); err != nil {
			return fmt.Errorf("actor %s auth credential: %w", actor.Name, err)
		}
		if err := validateData(actor.AuthInfo); err != nil {
			return fmt.Errorf("actor %s auth info: %w", actor.Name, err)
		}
		if err := validateService(actor.AuthService); err != nil {
			return fmt.Errorf("actor %s auth: %w", actor.Name, err)
		}
		if err := validateMethod("actor "+actor.Name+" auth method", actor.AuthMethod); err != nil {
			return err
		}
	}
	if actor.PermEnabled {
		if actor.PermService == nil || actor.PermMethod == nil {
			return fmt.Errorf("actor %s has incomplete permission support", actor.Name)
		}
	}
	if actor.PermService != nil {
		if err := validateService(actor.PermService); err != nil {
			return fmt.Errorf("actor %s permission: %w", actor.Name, err)
		}
	}
	if actor.PermMethod != nil {
		if err := validateMethod("actor "+actor.Name+" permission method", actor.PermMethod); err != nil {
			return err
		}
	}
	return nil
}

func validateResource(resource *model.Resource) error {
	if resource == nil {
		return fmt.Errorf("generated model contains nil resource")
	}
	if err := validateResourceChecks("resource "+resource.Name, resource.Checks); err != nil {
		return err
	}
	for _, action := range resource.Actions {
		if action == nil {
			return fmt.Errorf("resource %s contains a nil action", resource.Name)
		}
		if err := validateResourceChecks("resource "+resource.Name+" action "+action.Name, action.Checks); err != nil {
			return err
		}
	}
	if resource.CheckService != nil {
		if err := validateService(resource.CheckService); err != nil {
			return err
		}
	}
	return nil
}

func validateResourceChecks(owner string, checks []*model.ResourceCheck) error {
	for _, check := range checks {
		if check == nil {
			return fmt.Errorf("%s contains a nil check", owner)
		}
		if err := validateMethod(owner+" check "+check.Name, check.Method); err != nil {
			return err
		}
	}
	return nil
}

func validateData(data *model.Data) error {
	if data == nil {
		return fmt.Errorf("generated model contains nil data")
	}
	for _, member := range data.Members {
		if member == nil {
			return fmt.Errorf("data %s contains a nil member", data.Name)
		}
		if err := validateModelType(member.Type); err != nil {
			return fmt.Errorf("data %s member %s: %w", data.Name, member.Name, err)
		}
	}
	return nil
}

func validateService(service *model.Service) error {
	if service == nil {
		return fmt.Errorf("generated model contains nil service")
	}
	if err := validateAuthMode(service.Auth); err != nil {
		return fmt.Errorf("service %s: %w", service.Name, err)
	}
	if err := validatePermissionExpr(service.Require); err != nil {
		return fmt.Errorf("service %s: %w", service.Name, err)
	}
	if err := validateAudiences("service "+service.Name, service.Audiences); err != nil {
		return err
	}
	for _, method := range service.Methods {
		if method == nil {
			return fmt.Errorf("service %s contains a nil method", service.Name)
		}
		if err := validateMethod("service "+service.Name+" method "+method.Name, method); err != nil {
			return err
		}
	}
	return nil
}

func validateMethod(owner string, method *model.Method) error {
	if method == nil {
		return fmt.Errorf("%s is nil", owner)
	}
	if err := validateAuthMode(method.Auth); err != nil {
		return fmt.Errorf("%s: %w", owner, err)
	}
	if err := validatePermissionExpr(method.Require); err != nil {
		return fmt.Errorf("%s: %w", owner, err)
	}
	if err := validateArguments(owner, method.Arguments, method.ArgumentsData); err != nil {
		return err
	}
	if method.ResultType != nil {
		if err := validateModelType(method.ResultType); err != nil {
			return fmt.Errorf("%s result: %w", owner, err)
		}
	}
	return nil
}

func validateAudiences(owner string, audiences []*model.ActorAudience) error {
	for _, audience := range audiences {
		if audience == nil {
			return fmt.Errorf("%s contains a nil audience", owner)
		}
		if audience.Actor == "" {
			return fmt.Errorf("%s contains an audience without an actor", owner)
		}
		if audience.Via != "" {
			if err := validateActorVia(audience.Via); err != nil {
				return fmt.Errorf("%s: %w", owner, err)
			}
		}
	}
	return nil
}

func validateActorVia(via string) error {
	switch model.ActorViaKind(via) {
	case model.ActorViaClient, model.ActorViaAgent, model.ActorViaOpenAPI:
		return nil
	default:
		return fmt.Errorf("unsupported actor via %q", via)
	}
}

func validateArguments(owner string, arguments []*model.Argument, data *model.Data) error {
	members := map[string]bool{}
	if data != nil {
		if err := validateData(data); err != nil {
			return fmt.Errorf("%s arguments: %w", owner, err)
		}
		for _, member := range data.Members {
			members[member.Name] = true
		}
	}
	for _, argument := range arguments {
		if argument == nil {
			return fmt.Errorf("%s contains a nil argument", owner)
		}
		if err := validateModelType(argument.Type); err != nil {
			return fmt.Errorf("%s argument %s: %w", owner, argument.Name, err)
		}
		if data != nil && !members[argument.Name] {
			return fmt.Errorf("%s argument member %s not found", owner, argument.Name)
		}
	}
	return nil
}

func validateModelType(type_ *model.Type) error {
	if type_ == nil {
		return fmt.Errorf("type is nil")
	}
	return WalkType(type_, func(current *model.Type) error {
		switch current.Kind {
		case model.TypeKindScalar:
			if current.Scalar < model.ScalarInt || current.Scalar > model.ScalarJSON {
				return fmt.Errorf("unsupported scalar %s", current.Scalar.Name())
			}
		case model.TypeKindSkelPermissionCode:
		case model.TypeKindList:
			if current.List == nil || current.List.Value == nil {
				return fmt.Errorf("list metadata is nil")
			}
		case model.TypeKindMap:
			if current.Map == nil || current.Map.Key == nil || current.Map.Value == nil {
				return fmt.Errorf("map metadata is nil")
			}
		case model.TypeKindEnum:
			if current.Enum == nil {
				return fmt.Errorf("enum metadata is nil")
			}
		case model.TypeKindData:
			if current.Data == nil {
				return fmt.Errorf("data metadata is nil")
			}
			switch current.Data.Kind {
			case model.DataKindData, model.DataKindConfig, model.DataKindEvent:
			default:
				return fmt.Errorf("referenced data %s has unsupported kind %q", current.Data.Name, current.Data.Kind)
			}
		case model.TypeKindTypeParameter:
			if current.TypeParameter == nil {
				return fmt.Errorf("type parameter metadata is nil")
			}
		default:
			return fmt.Errorf("unsupported type kind %d", current.Kind)
		}
		return nil
	})
}

func validateAuthMode(mode model.AuthMode) error {
	switch mode {
	case "", model.AuthModeUnset, model.AuthModeAuth, model.AuthModeNoAuth:
		return nil
	default:
		return fmt.Errorf("unsupported auth mode %q", mode)
	}
}

func validatePermissionExpr(require *model.PermissionRequire) error {
	if require == nil || require.Expr == nil {
		return nil
	}
	var validate func(*model.PermissionExpr) error
	validate = func(expr *model.PermissionExpr) error {
		if expr == nil {
			return fmt.Errorf("permission expression is nil")
		}
		switch expr.Mode {
		case model.PermissionRequireModeCode:
		case model.PermissionRequireModeCheck:
			if expr.Check == nil {
				return fmt.Errorf("permission check invocation is nil")
			}
			for _, argument := range expr.Check.Arguments {
				if argument == nil {
					return fmt.Errorf("permission check contains a nil argument")
				}
				if err := validateModelType(argument.Type); err != nil {
					return fmt.Errorf("permission check argument %s: %w", argument.Name, err)
				}
			}
		case model.PermissionRequireModeAll, model.PermissionRequireModeAny:
			for _, child := range expr.Children {
				if err := validate(child); err != nil {
					return err
				}
			}
		default:
			return fmt.Errorf("unsupported permission require mode %q", expr.Mode)
		}
		return nil
	}
	return validate(require.Expr)
}
