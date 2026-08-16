package schema

import "fmt"

func validateRequirement(value *Requirement) error {
	if value == nil {
		return nil
	}
	switch value.Mode {
	case RequirementModeCode:
		if value.Code == "" {
			return fmt.Errorf("code permission requirement has no code")
		}
		if value.Check != nil || len(value.Children) != 0 {
			return fmt.Errorf("code permission requirement contains unrelated fields")
		}
	case RequirementModeReference:
		if value.Check == nil || value.Check.Resource == "" || value.Check.Action == "" {
			return fmt.Errorf("reference permission requirement is incomplete")
		}
		if value.Code != "" || len(value.Children) != 0 {
			return fmt.Errorf("reference permission requirement contains unrelated fields")
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
	case RequirementModeCheck:
		if value.Check == nil || value.Check.Resource == "" || value.Check.Check == "" {
			return fmt.Errorf("check permission requirement is incomplete")
		}
		if value.Code != "" || len(value.Children) != 0 {
			return fmt.Errorf("check permission requirement contains unrelated fields")
		}
		for _, argument := range value.Check.Arguments {
			if argument == nil || argument.Name == "" {
				return fmt.Errorf("check permission requirement has an unnamed argument")
			}
			if err := validateType(argument.Type); err != nil {
				return err
			}
		}
	case RequirementModeAll, RequirementModeAny:
		if len(value.Children) == 0 {
			return fmt.Errorf("%s permission requirement has no children", value.Mode)
		}
		if value.Code != "" || value.Check != nil {
			return fmt.Errorf("%s permission requirement contains unrelated fields", value.Mode)
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
	case TypeKindScalar, TypeKindEnum, TypeKindData, TypeKindTypeParameter, TypeKindImportedReference:
		if value.Name == "" {
			return fmt.Errorf("%s type has no name", value.Kind)
		}
		if value.Element != nil || value.Key != nil || value.Value != nil {
			return fmt.Errorf("%s type contains unrelated fields", value.Kind)
		}
		if value.Kind != TypeKindData && value.Kind != TypeKindImportedReference && len(value.Arguments) != 0 {
			return fmt.Errorf("%s type contains type arguments", value.Kind)
		}
	case TypeKindList:
		if value.Name != "" || len(value.Arguments) != 0 || value.Key != nil || value.Value != nil {
			return fmt.Errorf("list type contains unrelated fields")
		}
		if err := validateType(value.Element); err != nil {
			return fmt.Errorf("list element: %w", err)
		}
	case TypeKindMap:
		if value.Name != "" || len(value.Arguments) != 0 || value.Element != nil {
			return fmt.Errorf("map type contains unrelated fields")
		}
		if err := validateType(value.Key); err != nil {
			return fmt.Errorf("map key: %w", err)
		}
		if err := validateType(value.Value); err != nil {
			return fmt.Errorf("map value: %w", err)
		}
	case TypeKindPermissionCode:
		if value.Name != "" || len(value.Arguments) != 0 || value.Element != nil || value.Key != nil || value.Value != nil {
			return fmt.Errorf("permissionCode type contains unrelated fields")
		}
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
