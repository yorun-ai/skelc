package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/model"
)

type _CloneDataKey struct {
	data      *model.Data
	typePlain string
}

type _CloneBuilder struct {
	imports             _ImportSet
	typeParameterCloner map[*model.TypeParameter]string
	nextVariable        int
}

func buildDataClone(parsed *model.Data, data *Data) {
	// Generated packages are fully managed by skelc. Clone generation only
	// reasons about declarations in the Skel model; handwritten Go files and
	// any methods or codec behavior they add are outside the supported contract.
	data.CloneMethodName = dataCloneMethodName(parsed)
	typeParameterCloners := make(map[*model.TypeParameter]string, len(parsed.TypeParameters))
	parameters := make([]*_GoParameter, 0, len(parsed.TypeParameters))
	for _, parameter := range parsed.TypeParameters {
		clonerName := "clone" + parameter.Name
		typeParameterCloners[parameter] = clonerName
		parameters = append(parameters, goParameter(clonerName, fmt.Sprintf("func(%s) %s", parameter.Name, parameter.Name)))
	}
	if !cloneDataSupported(parsed, typeParameterCloners) {
		return
	}

	builder := &_CloneBuilder{
		imports:             newImportSet(),
		typeParameterCloner: typeParameterCloners,
	}
	data.CloneBlock = goBlock()
	for _, member := range parsed.Members {
		memberName := nameutil.ToCamel(member.Name)
		data.CloneBlock.append(builder.cloneAssignStatements(
			member.Type,
			"v."+memberName,
			"cloned."+memberName,
		)...)
	}
	data.Clone = true
	data.CloneParameters = parameters
	data.CloneImports = builder.imports.sortedValues()
}

func buildMethodClones(parsed *model.Method, method *ServiceMethod) {
	builder := &_CloneBuilder{
		imports:             newImportSet(),
		typeParameterCloner: map[*model.TypeParameter]string{},
	}
	if method.ArgumentsData != nil && cloneTypesSupported(methodArgumentTypes(parsed), nil) {
		method.CloneArguments = builder.buildArgumentsClone(parsed, method)
	}
	if method.ResultType != nil && cloneTypesSupported([]*model.Type{parsed.ResultType}, nil) {
		method.CloneResult = builder.buildResultClone(parsed.ResultType, method.ResultType)
	}
	method.CloneImports = builder.imports.sortedValues()
}

func methodArgumentTypes(method *model.Method) []*model.Type {
	types := make([]*model.Type, 0, len(method.Arguments))
	for _, argument := range method.Arguments {
		types = append(types, argument.Type)
	}
	return types
}

func cloneDataSupported(data *model.Data, typeParameterCloners map[*model.TypeParameter]string) bool {
	if data == nil || dataCloneMethodConflicts(data) {
		return false
	}
	active := map[*model.Data]bool{data: true}
	completed := map[_CloneDataKey]bool{}
	for _, member := range data.Members {
		if member == nil || !cloneTypeSupported(member.Type, completed, active, typeParameterCloners) {
			return false
		}
	}
	return true
}

func dataCloneMethodConflicts(data *model.Data) bool {
	methodName := dataCloneMethodName(data)
	for _, member := range data.Members {
		if member != nil && nameutil.ToCamel(member.Name) == methodName {
			return true
		}
	}
	return false
}

func dataCloneMethodName(data *model.Data) string {
	if len(data.TypeParameters) > 0 {
		return "CloneBy"
	}
	return "Clone"
}

func cloneTypesSupported(types []*model.Type, typeParameterCloners map[*model.TypeParameter]string) bool {
	completed := map[_CloneDataKey]bool{}
	for _, type_ := range types {
		if !cloneTypeSupported(resolveCloneType(type_, nil), completed, map[*model.Data]bool{}, typeParameterCloners) {
			return false
		}
	}
	return true
}

func cloneTypeSupported(
	type_ *model.Type,
	completed map[_CloneDataKey]bool,
	active map[*model.Data]bool,
	typeParameterCloners map[*model.TypeParameter]string,
) bool {
	if type_ == nil {
		return false
	}
	switch type_.Kind {
	case model.TypeKindScalar, model.TypeKindSkelPermissionCode, model.TypeKindEnum:
		return true
	case model.TypeKindList:
		return type_.List != nil && cloneTypeSupported(type_.List.Value, completed, active, typeParameterCloners)
	case model.TypeKindMap:
		return type_.Map != nil &&
			cloneTypeSupported(type_.Map.Key, completed, active, typeParameterCloners) &&
			cloneTypeSupported(type_.Map.Value, completed, active, typeParameterCloners)
	case model.TypeKindData:
		if type_.Data == nil ||
			(type_.Data.Kind != "" && type_.Data.Kind != model.DataKindData) ||
			dataCloneMethodConflicts(type_.Data) {
			return false
		}
		if type_.ExternalDomain != "" || type_.ExternalImportPath != "" {
			// TODO: After generated packages have had several skelc release cycles to
			// adopt Clone and CloneBy, assume imported data provides those methods and
			// include it in typed clone hooks instead of using serialization fallback.
			return false
		}
		plain := castType(withoutCloneNullable(type_)).Plain
		key := _CloneDataKey{data: type_.Data, typePlain: plain}
		if active[type_.Data] {
			return false
		}
		if completed[key] {
			return true
		}
		active[type_.Data] = true
		defer delete(active, type_.Data)
		substitutions := cloneDataSubstitutions(type_)
		for _, member := range type_.Data.Members {
			if member == nil || !cloneTypeSupported(
				resolveCloneType(member.Type, substitutions),
				completed,
				active,
				typeParameterCloners,
			) {
				return false
			}
		}
		completed[key] = true
		return true
	case model.TypeKindTypeParameter:
		return typeParameterCloners[type_.TypeParameter] != ""
	default:
		return false
	}
}

func (b *_CloneBuilder) buildArgumentsClone(parsed *model.Method, method *ServiceMethod) *_GoFunction {
	body := goBlock(
		goAssignmentStatement("source", ":=", goRaw(fmt.Sprintf("value.(*%s)", method.ArgumentsData.Name))),
		goAssignmentStatement("cloned", ":=", goRaw("*source")),
	)
	for index, argument := range parsed.Arguments {
		body.append(b.cloneAssignStatements(
			resolveCloneType(argument.Type, nil),
			"source."+method.Arguments[index].MemberName,
			"cloned."+method.Arguments[index].MemberName,
		)...)
	}
	body.append(goReturnStatement(goRaw("&cloned")))
	return goFunction([]*_GoParameter{goParameter("value", "any")}, "any", body)
}

func (b *_CloneBuilder) buildResultClone(parsedType *model.Type, resultType *Type) *_GoFunction {
	body := goBlock(
		goAssignmentStatement("source", ":=", goRaw(fmt.Sprintf("value.(%s)", resultType.Plain))),
		goAssignmentStatement("cloned", ":=", goRaw("source")),
	)
	body.append(b.cloneAssignStatements(resolveCloneType(parsedType, nil), "source", "cloned")...)
	body.append(goReturnStatement(goRaw("cloned")))
	return goFunction([]*_GoParameter{goParameter("value", "any")}, "any", body)
}

func (b *_CloneBuilder) cloneAssignStatements(type_ *model.Type, source string, target string) []*_GoStatement {
	if cloneTypeUsesNullablePointer(type_) {
		valueName := b.variableName("clonedValue")
		thenBlock := goBlock(
			goAssignmentStatement(valueName, ":=", goRaw("*"+source)),
		)
		thenBlock.append(b.cloneAssignStatements(withoutCloneNullable(type_), "(*"+source+")", valueName)...)
		thenBlock.append(goAssignmentStatement(target, "=", goRaw("&"+valueName)))
		return []*_GoStatement{goIfStatement(nil, goRaw(source+" != nil"), thenBlock, nil)}
	}

	switch type_.Kind {
	case model.TypeKindScalar:
		if type_.Scalar == model.ScalarBinary {
			return b.cloneSliceAssignStatements(type_, nil, source, target)
		}
	case model.TypeKindList:
		return b.cloneSliceAssignStatements(type_, type_.List.Value, source, target)
	case model.TypeKindMap:
		b.imports.add(&Import{Path: "maps"})
		keyName := b.variableName("key")
		itemName := b.variableName("item")
		clonedItemName := b.variableName("clonedItem")
		itemStatements := b.cloneAssignStatements(type_.Map.Value, itemName, clonedItemName)
		statements := []*_GoStatement{
			goAssignmentStatement(target, "=", goCall("maps.Clone", goRaw(source))),
		}
		if len(itemStatements) > 0 {
			body := goBlock(
				goAssignmentStatement(clonedItemName, ":=", goRaw(itemName)),
			)
			body.append(itemStatements...)
			body.append(goAssignmentStatement(
				fmt.Sprintf("%s[%s]", target, keyName),
				"=",
				goRaw(clonedItemName),
			))
			statements = append(statements, goRangeStatement(
				[]string{keyName, itemName},
				goRaw(source),
				body,
			))
		}
		return statements
	case model.TypeKindData:
		return []*_GoStatement{goAssignmentStatement(target, "=", b.cloneDataExpression(type_, source))}
	case model.TypeKindTypeParameter:
		return []*_GoStatement{goAssignmentStatement(
			target,
			"=",
			goCall(b.typeParameterCloner[type_.TypeParameter], goRaw(source)),
		)}
	}
	return nil
}

func (b *_CloneBuilder) cloneSliceAssignStatements(type_ *model.Type, elementType *model.Type, source string, target string) []*_GoStatement {
	castedType := castType(withoutCloneNullable(type_))
	thenBlock := goBlock(goAssignmentStatement(target, "=", goRaw("nil")))
	elseBlock := goBlock(goAssignmentStatement(
		target,
		"=",
		goCall("make", goRaw(castedType.Plain), goCall("len", goRaw(source))),
	))
	if elementType == nil {
		elseBlock.append(goExpressionStatement(goCall("copy", goRaw(target), goRaw(source))))
		return []*_GoStatement{goIfStatement(nil, goRaw(source+" == nil"), thenBlock, elseBlock)}
	}

	indexName := b.variableName("index")
	elementStatements := b.cloneAssignStatements(
		elementType,
		source+"["+indexName+"]",
		target+"["+indexName+"]",
	)
	if len(elementStatements) == 0 {
		elseBlock.append(goExpressionStatement(goCall("copy", goRaw(target), goRaw(source))))
	} else {
		elseBlock.append(goRangeStatement(
			[]string{indexName},
			goRaw(source),
			goBlock(elementStatements...),
		))
	}
	return []*_GoStatement{goIfStatement(nil, goRaw(source+" == nil"), thenBlock, elseBlock)}
}

func (b *_CloneBuilder) cloneDataExpression(type_ *model.Type, source string) *_GoExpression {
	nonNullableType := withoutCloneNullable(type_)
	castedType := castType(nonNullableType)
	b.imports.addMany(castedType.Imports)
	if len(nonNullableType.TypeArguments) == 0 {
		return goCall(source + ".Clone")
	}

	cloners := make([]*_GoExpression, 0, len(nonNullableType.TypeArguments))
	for _, argument := range nonNullableType.TypeArguments {
		cloners = append(cloners, b.cloneFunctionExpression(argument))
	}
	return goCall(source+".CloneBy", cloners...)
}

func (b *_CloneBuilder) cloneFunctionExpression(type_ *model.Type) *_GoExpression {
	if type_.Kind == model.TypeKindTypeParameter {
		return goRaw(b.typeParameterCloner[type_.TypeParameter])
	}

	castedType := castType(type_)
	b.imports.addMany(castedType.Imports)
	if type_.Kind == model.TypeKindData && !type_.Nullable {
		return goFunctionExpression(goFunction(
			[]*_GoParameter{goParameter("value", castedType.Plain)},
			castedType.Plain,
			goBlock(goReturnStatement(b.cloneDataExpression(type_, "value"))),
		))
	}
	cloneStatements := b.cloneAssignStatements(type_, "value", "cloned")
	if len(cloneStatements) == 0 {
		return goFunctionExpression(goFunction(
			[]*_GoParameter{goParameter("value", castedType.Plain)},
			castedType.Plain,
			goBlock(goReturnStatement(goRaw("value"))),
		))
	}
	body := goBlock(goAssignmentStatement("cloned", ":=", goRaw("value")))
	body.append(cloneStatements...)
	body.append(goReturnStatement(goRaw("cloned")))
	return goFunctionExpression(goFunction(
		[]*_GoParameter{goParameter("value", castedType.Plain)},
		castedType.Plain,
		body,
	))
}

func (b *_CloneBuilder) variableName(prefix string) string {
	name := fmt.Sprintf("%s%d", prefix, b.nextVariable)
	b.nextVariable++
	return name
}

func cloneTypeUsesNullablePointer(type_ *model.Type) bool {
	if type_ == nil || !type_.Nullable {
		return false
	}
	switch type_.Kind {
	case model.TypeKindScalar, model.TypeKindEnum, model.TypeKindData:
		return true
	default:
		return false
	}
}

func withoutCloneNullable(type_ *model.Type) *model.Type {
	cloned := *type_
	cloned.Nullable = false
	return &cloned
}

func cloneDataSubstitutions(type_ *model.Type) map[*model.TypeParameter]*model.Type {
	substitutions := make(map[*model.TypeParameter]*model.Type, len(type_.Data.TypeParameters))
	for index, parameter := range type_.Data.TypeParameters {
		if index < len(type_.TypeArguments) {
			substitutions[parameter] = type_.TypeArguments[index]
		}
	}
	return substitutions
}

func resolveCloneType(type_ *model.Type, substitutions map[*model.TypeParameter]*model.Type) *model.Type {
	if type_ == nil {
		return nil
	}
	if type_.Kind == model.TypeKindTypeParameter {
		if replacement := substitutions[type_.TypeParameter]; replacement != nil {
			return resolveCloneType(replacement, substitutions)
		}
	}
	resolved := *type_
	switch type_.Kind {
	case model.TypeKindList:
		resolved.List = &model.ListType{Value: resolveCloneType(type_.List.Value, substitutions)}
	case model.TypeKindMap:
		resolved.Map = &model.MapType{
			Key:   resolveCloneType(type_.Map.Key, substitutions),
			Value: resolveCloneType(type_.Map.Value, substitutions),
		}
	case model.TypeKindData:
		resolved.TypeArguments = make([]*model.Type, 0, len(type_.TypeArguments))
		for _, argument := range type_.TypeArguments {
			resolved.TypeArguments = append(resolved.TypeArguments, resolveCloneType(argument, substitutions))
		}
	}
	return &resolved
}
