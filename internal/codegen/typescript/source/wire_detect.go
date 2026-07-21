package source

import "go.yorun.ai/skelc/model"

func methodArgumentsContainBinary(method *model.Method) bool {
	for _, argument := range method.Arguments {
		if typeContainsBinary(argument.Type, nil, map[*model.Data]bool{}) {
			return true
		}
	}
	return false
}

func methodResultContainsBinary(method *model.Method) bool {
	return typeContainsBinary(method.ResultType, nil, map[*model.Data]bool{})
}

func typeContainsBinary(
	type_ *model.Type,
	typeArguments map[*model.TypeParameter]*model.Type,
	visiting map[*model.Data]bool,
) bool {
	if type_ == nil {
		return false
	}

	switch type_.Kind {
	case model.TypeKindScalar:
		return type_.Scalar == model.ScalarBinary
	case model.TypeKindList:
		return typeContainsBinary(type_.List.Value, typeArguments, visiting)
	case model.TypeKindMap:
		return typeContainsBinary(type_.Map.Value, typeArguments, visiting)
	case model.TypeKindTypeParameter:
		return typeContainsBinary(typeArguments[type_.TypeParameter], typeArguments, visiting)
	case model.TypeKindData:
		if visiting[type_.Data] {
			return false
		}
		visiting[type_.Data] = true
		defer delete(visiting, type_.Data)

		nestedArguments := cloneTypeArguments(typeArguments)
		for index, parameter := range type_.Data.TypeParameters {
			if index < len(type_.TypeArguments) {
				nestedArguments[parameter] = resolveTypeArgument(type_.TypeArguments[index], typeArguments)
			}
		}
		for _, member := range type_.Data.Members {
			if typeContainsBinary(member.Type, nestedArguments, visiting) {
				return true
			}
		}
	}
	return false
}

func cloneTypeArguments(source map[*model.TypeParameter]*model.Type) map[*model.TypeParameter]*model.Type {
	cloned := make(map[*model.TypeParameter]*model.Type, len(source))
	for parameter, type_ := range source {
		cloned[parameter] = type_
	}
	return cloned
}

func resolveTypeArgument(
	type_ *model.Type,
	typeArguments map[*model.TypeParameter]*model.Type,
) *model.Type {
	if type_ != nil && type_.Kind == model.TypeKindTypeParameter {
		if resolved := typeArguments[type_.TypeParameter]; resolved != nil {
			return resolved
		}
	}
	return type_
}
