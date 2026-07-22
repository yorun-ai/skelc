package analyzer

import (
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
	"reflect"
	"strings"
)

func resolveMethodArgumentJsonPath(method *model.Method, path string) *model.Type {
	parts := parsePermissionCheckJsonPath(path)
	checkutil.Check(len(parts) > 0, "empty require check argument path")
	for _, arg := range method.Arguments {
		if arg.Name != parts[0].Name {
			continue
		}
		return resolveJsonPathPartType(arg.Type, parts[0], parts[1:], path)
	}
	checkutil.Failf(`require check argument path %s references undefined input argument "%s"`, path, parts[0].Name)
	panic("unreachable")
}

type _PermissionCheckPathPart struct {
	Name     string
	Wildcard bool
}

func parsePermissionCheckJsonPath(path string) []_PermissionCheckPathPart {
	rawParts := strings.Split(path, ".")
	parts := make([]_PermissionCheckPathPart, 0, len(rawParts))
	wildcardCount := 0
	for _, rawPart := range rawParts {
		checkutil.Check(rawPart != "", "require check argument path %s contains empty field", path)
		part := _PermissionCheckPathPart{Name: rawPart}
		if strings.HasSuffix(rawPart, "[*]") {
			part.Name = strings.TrimSuffix(rawPart, "[*]")
			part.Wildcard = true
			wildcardCount++
		}
		checkutil.Check(part.Name != "", "require check argument path %s contains empty field", path)
		checkutil.Check(!strings.ContainsAny(part.Name, "[]"),
			"require check argument path %s only supports field cascade and one [*]", path)
		checkutil.Check(wildcardCount <= 1,
			"require check argument path %s supports at most one [*]", path)
		parts = append(parts, part)
	}
	checkutil.Check(!parts[len(parts)-1].Wildcard,
		"require check argument path %s cannot end with [*], use the list field path directly", path)
	return parts
}

func resolveJsonPathType(type_ *model.Type, parts []_PermissionCheckPathPart, fullPath string) *model.Type {
	if len(parts) == 0 {
		return type_
	}
	checkutil.Check(type_.Kind == model.TypeKindData, "require check argument path %s cannot select member on %s", fullPath, type_.Name())
	for _, member := range type_.Data.Members {
		if member.Name == parts[0].Name {
			return resolveJsonPathPartType(member.Type, parts[0], parts[1:], fullPath)
		}
	}
	checkutil.Failf(`require check argument path %s references undefined data member "%s"`, fullPath, parts[0].Name)
	panic("unreachable")
}

func resolveJsonPathPartType(type_ *model.Type, part _PermissionCheckPathPart, remainingParts []_PermissionCheckPathPart, fullPath string) *model.Type {
	if !part.Wildcard {
		return resolveJsonPathType(type_, remainingParts, fullPath)
	}
	checkutil.Check(type_.Kind == model.TypeKindList,
		"require check argument path %s can only use [*] on list, got %s", fullPath, type_.Name())
	valueType := resolveJsonPathType(type_.List.Value, remainingParts, fullPath)
	return &model.Type{
		Kind: model.TypeKindList,
		List: &model.ListType{Value: valueType},
	}
}

func typeEqual(a *model.Type, b *model.Type) bool {
	if a.Kind != b.Kind || a.Nullable != b.Nullable || a.Scalar != b.Scalar || a.SkelName != b.SkelName {
		return false
	}
	switch a.Kind {
	case model.TypeKindList:
		return typeEqual(a.List.Value, b.List.Value)
	case model.TypeKindMap:
		return typeEqual(a.Map.Key, b.Map.Key) && typeEqual(a.Map.Value, b.Map.Value)
	default:
		return reflect.DeepEqual(typeArgumentNames(a), typeArgumentNames(b))
	}
}

func typeArgumentNames(type_ *model.Type) []string {
	values := make([]string, 0, len(type_.TypeArguments))
	for _, argument := range type_.TypeArguments {
		values = append(values, argument.Name())
	}
	return values
}
