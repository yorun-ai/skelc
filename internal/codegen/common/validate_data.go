package common

import (
	"fmt"

	"go.yorun.ai/skelc/model"
)

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
