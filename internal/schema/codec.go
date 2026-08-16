package schema

import (
	"encoding/json"
	"fmt"
	"io"
)

func Encode(writer io.Writer, document *Document) error {
	if err := Validate(document); err != nil {
		return err
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(document); err != nil {
		return fmt.Errorf("encode schema: %w", err)
	}
	return nil
}

func Decode(reader io.Reader) (*Document, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	document := new(Document)
	if err := decoder.Decode(document); err != nil {
		return nil, fmt.Errorf("decode schema: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("decode schema: unexpected trailing JSON value")
		}
		return nil, fmt.Errorf("decode schema: %w", err)
	}
	if err := Validate(document); err != nil {
		return nil, err
	}
	return document, nil
}

func Validate(document *Document) error {
	if document == nil {
		return fmt.Errorf("schema is required")
	}
	if document.Format != Format {
		return fmt.Errorf("unsupported schema format %q", document.Format)
	}
	if document.FormatVersion != FormatVersion {
		return fmt.Errorf("unsupported schema format version %d", document.FormatVersion)
	}
	if document.Domain == "" {
		return fmt.Errorf("schema domain is required")
	}
	if document.Declarations == nil {
		return fmt.Errorf("schema declarations are required")
	}
	seen := map[string]bool{}
	for index, declaration := range document.Declarations {
		if declaration == nil {
			return fmt.Errorf("schema declaration %d is null", index)
		}
		if declaration.Name == "" || declaration.SkelName == "" || declaration.Kind == "" {
			return fmt.Errorf("schema declaration %d has an empty name or type", index)
		}
		key := declarationKey(declaration)
		if seen[key] {
			return fmt.Errorf("schema contains duplicated %s declaration %s", declaration.Kind, declaration.SkelName)
		}
		seen[key] = true
		if err := validateDeclarationBody(declaration); err != nil {
			return fmt.Errorf("schema declaration %s: %w", declaration.SkelName, err)
		}
	}
	return nil
}

func validateDeclarationBody(declaration *Declaration) error {
	present := 0
	for _, bodyPresent := range []bool{
		declaration.Enum != nil,
		declaration.Data != nil,
		declaration.Actor != nil,
		declaration.Resource != nil,
		declaration.Service != nil,
		declaration.Web != nil,
		declaration.Task != nil,
	} {
		if bodyPresent {
			present++
		}
	}
	if present != 1 {
		return fmt.Errorf("expected exactly one declaration body")
	}
	valid := false
	switch declaration.Kind {
	case "enum":
		valid = declaration.Enum != nil
	case "data", "config", "event":
		valid = declaration.Data != nil
	case "actor":
		valid = declaration.Actor != nil
	case "resource":
		valid = declaration.Resource != nil
	case "service":
		valid = declaration.Service != nil
	case "web":
		valid = declaration.Web != nil
	case "task":
		valid = declaration.Task != nil
	}
	if !valid {
		return fmt.Errorf("declaration body does not match type %q", declaration.Kind)
	}
	switch declaration.Kind {
	case "enum":
		return validateEnum(declaration.Enum)
	case "data", "config", "event":
		return validateData(declaration.Data)
	case "actor":
		return validateActor(declaration.Actor)
	case "resource":
		return validateResource(declaration.Resource)
	case "service":
		return validateService(declaration.Service)
	case "web":
		return validateAudiences(declaration.Web.Audiences)
	case "task":
		return validateTask(declaration.Task)
	default:
		return nil
	}
}

func validateEnum(value *EnumSchema) error {
	seen := map[string]bool{}
	for index, item := range value.Items {
		if item == nil || item.Name == "" {
			return fmt.Errorf("enum item %d is null or unnamed", index)
		}
		if seen[item.Name] {
			return fmt.Errorf("duplicated enum item %s", item.Name)
		}
		seen[item.Name] = true
	}
	return nil
}

func validateData(value *DataSchema) error {
	return validateMembers(value.Members)
}

func validateMembers(values []*Member) error {
	seen := map[string]bool{}
	for index, member := range values {
		if member == nil || member.Name == "" {
			return fmt.Errorf("member %d is null or unnamed", index)
		}
		if seen[member.Name] {
			return fmt.Errorf("duplicated member %s", member.Name)
		}
		seen[member.Name] = true
		if err := validateType(member.Type); err != nil {
			return fmt.Errorf("member %s: %w", member.Name, err)
		}
	}
	return nil
}

func validateActor(value *ActorSchema) error {
	seen := map[string]bool{}
	for index, via := range value.Vias {
		if via == nil || via.Name == "" {
			return fmt.Errorf("actor via %d is null or unnamed", index)
		}
		if seen[via.Name] {
			return fmt.Errorf("duplicated actor via %s", via.Name)
		}
		seen[via.Name] = true
	}
	if value.AuthEnabled {
		if value.AuthCredential == nil || value.AuthInfo == nil {
			return fmt.Errorf("authenticated actor requires credential and info schemas")
		}
		if err := validateData(value.AuthCredential); err != nil {
			return fmt.Errorf("actor credential: %w", err)
		}
		if err := validateData(value.AuthInfo); err != nil {
			return fmt.Errorf("actor info: %w", err)
		}
	} else if value.AuthCredential != nil || value.AuthInfo != nil {
		return fmt.Errorf("actor without authentication contains credential or info schema")
	}
	return nil
}

func validateResource(value *ResourceSchema) error {
	if err := validateResourceChecks(value.Checks); err != nil {
		return err
	}
	seen := map[string]bool{}
	for index, action := range value.Actions {
		if action == nil || action.Name == "" {
			return fmt.Errorf("resource action %d is null or unnamed", index)
		}
		if seen[action.Name] {
			return fmt.Errorf("duplicated resource action %s", action.Name)
		}
		seen[action.Name] = true
		if err := validateResourceChecks(action.Checks); err != nil {
			return fmt.Errorf("resource action %s: %w", action.Name, err)
		}
	}
	return nil
}

func validateResourceChecks(values []*ResourceCheck) error {
	seen := map[string]bool{}
	for index, check := range values {
		if check == nil || check.Name == "" {
			return fmt.Errorf("resource check %d is null or unnamed", index)
		}
		if seen[check.Name] {
			return fmt.Errorf("duplicated resource check %s", check.Name)
		}
		seen[check.Name] = true
		if err := validateArguments(check.Arguments); err != nil {
			return fmt.Errorf("resource check %s: %w", check.Name, err)
		}
	}
	return nil
}

func validateService(value *ServiceSchema) error {
	if err := validateAudiences(value.Audiences); err != nil {
		return err
	}
	if err := validateRequirement(value.Require); err != nil {
		return err
	}
	seen := map[string]bool{}
	for index, method := range value.Methods {
		if method == nil || method.Name == "" || method.SkelName == "" {
			return fmt.Errorf("service method %d is null or unnamed", index)
		}
		if seen[method.Name] {
			return fmt.Errorf("duplicated service method %s", method.Name)
		}
		seen[method.Name] = true
		if err := validateArguments(method.Arguments); err != nil {
			return fmt.Errorf("service method %s: %w", method.Name, err)
		}
		if err := validateTypeOptional(method.Result); err != nil {
			return fmt.Errorf("service method %s result: %w", method.Name, err)
		}
		if err := validateRequirement(method.Require); err != nil {
			return fmt.Errorf("service method %s: %w", method.Name, err)
		}
	}
	return nil
}

func validateArguments(values []*Argument) error {
	seen := map[string]bool{}
	for index, argument := range values {
		if argument == nil || argument.Name == "" {
			return fmt.Errorf("argument %d is null or unnamed", index)
		}
		if seen[argument.Name] {
			return fmt.Errorf("duplicated argument %s", argument.Name)
		}
		seen[argument.Name] = true
		if err := validateType(argument.Type); err != nil {
			return fmt.Errorf("argument %s: %w", argument.Name, err)
		}
	}
	return nil
}

func validateAudiences(values []*Audience) error {
	seen := map[string]bool{}
	for index, audience := range values {
		if audience == nil || audience.Actor == "" {
			return fmt.Errorf("audience %d is null or unnamed", index)
		}
		key := audience.Actor + "\x00" + audience.Via
		if seen[key] {
			return fmt.Errorf("duplicated audience %s via %s", audience.Actor, audience.Via)
		}
		seen[key] = true
	}
	return nil
}

func validateTask(value *TaskSchema) error {
	seen := map[string]bool{}
	for index, trigger := range value.Triggers {
		if trigger == nil || trigger.Name == "" || trigger.SkelName == "" {
			return fmt.Errorf("task trigger %d is null or unnamed", index)
		}
		if seen[trigger.Name] {
			return fmt.Errorf("duplicated task trigger %s", trigger.Name)
		}
		seen[trigger.Name] = true
		if err := validateArguments(trigger.Arguments); err != nil {
			return fmt.Errorf("task trigger %s: %w", trigger.Name, err)
		}
	}
	return nil
}

func validateRequirement(value *Requirement) error {
	if value == nil {
		return nil
	}
	switch value.Mode {
	case "code":
		if value.Code == "" {
			return fmt.Errorf("code permission requirement has no code")
		}
	case "reference":
		if value.Check == nil || value.Check.Resource == "" || value.Check.Action == "" {
			return fmt.Errorf("reference permission requirement is incomplete")
		}
		for _, argument := range value.Check.Arguments {
			if argument == nil || argument.Name == "" && argument.JSONPath == "" {
				return fmt.Errorf("reference permission requirement has an unnamed argument")
			}
			if argument.Type != nil {
				if err := validateType(argument.Type); err != nil {
					return err
				}
			}
		}
	case "check":
		if value.Check == nil || value.Check.Resource == "" || value.Check.Check == "" {
			return fmt.Errorf("check permission requirement is incomplete")
		}
		for _, argument := range value.Check.Arguments {
			if argument == nil || argument.Name == "" {
				return fmt.Errorf("check permission requirement has an unnamed argument")
			}
			if err := validateType(argument.Type); err != nil {
				return err
			}
		}
	case "all", "any":
		if len(value.Children) == 0 {
			return fmt.Errorf("%s permission requirement has no children", value.Mode)
		}
		for _, child := range value.Children {
			if err := validateRequirement(child); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported permission requirement mode %q", value.Mode)
	}
	return nil
}

func validateTypeOptional(value *Type) error {
	if value == nil {
		return nil
	}
	return validateType(value)
}

func validateType(value *Type) error {
	if value == nil {
		return fmt.Errorf("type is required")
	}
	switch value.Kind {
	case "scalar", "enum", "data", "typeParameter", "importedReference":
		if value.Name == "" {
			return fmt.Errorf("%s type has no name", value.Kind)
		}
	case "list":
		if err := validateType(value.Element); err != nil {
			return fmt.Errorf("list element: %w", err)
		}
	case "map":
		if err := validateType(value.Key); err != nil {
			return fmt.Errorf("map key: %w", err)
		}
		if err := validateType(value.Value); err != nil {
			return fmt.Errorf("map value: %w", err)
		}
	case "permissionCode":
	default:
		return fmt.Errorf("unsupported type kind %q", value.Kind)
	}
	for index, argument := range value.Arguments {
		if err := validateType(argument); err != nil {
			return fmt.Errorf("type argument %d: %w", index, err)
		}
	}
	return nil
}
