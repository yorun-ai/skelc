package source

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/model"
	"go.yorun.ai/skelc/internal/skelmeta"
	"go.yorun.ai/skelc/internal/util/nameutil"
)

type _CloneDataKey struct {
	data      *model.Data
	typePlain string
}

type _CloneBuilder struct {
	imports             _ImportSet
	typeParameterCloner map[*model.TypeParameter]string
	activeImportedClone map[_CloneDataKey]string
	nextVariable        int
}

func newCloneBuilder(typeParameterCloners map[*model.TypeParameter]string) *_CloneBuilder {
	return &_CloneBuilder{
		imports:             newImportSet(),
		typeParameterCloner: typeParameterCloners,
		activeImportedClone: map[_CloneDataKey]string{},
	}
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

	builder := newCloneBuilder(typeParameterCloners)
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
	builder := newCloneBuilder(map[*model.TypeParameter]string{})
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
		return skelmeta.CloneByMethodName
	}
	return skelmeta.CloneMethodName
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
			(type_.Data.Kind != "" && type_.Data.Kind != model.DataKindData) {
			return false
		}
		if type_.ExternalDomain != "" || type_.ExternalImportPath != "" {
			// TODO(v0.13.0): Require imported generated data to provide Clone or
			// CloneBy and remove the v0.12 rolling-compatibility fallback.
			for _, argument := range type_.TypeArguments {
				if !cloneTypeSupported(argument, completed, active, typeParameterCloners) {
					return false
				}
			}
			return true
		}
		if dataCloneMethodConflicts(type_.Data) {
			return false
		}
		plain := castType(withoutCloneNullable(type_)).Plain
		key := _CloneDataKey{data: type_.Data, typePlain: plain}
		if active[type_.Data] {
			return true
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
	return b.cloneAssignStatementsWithSubstitutions(type_, nil, source, target)
}

func (b *_CloneBuilder) cloneAssignStatementsWithSubstitutions(
	type_ *model.Type,
	substitutions map[*model.TypeParameter]*model.Type,
	source string,
	target string,
) []*_GoStatement {
	if type_ == nil {
		return nil
	}
	if type_.Kind == model.TypeKindTypeParameter {
		if cloner := b.typeParameterCloner[type_.TypeParameter]; cloner != "" {
			return []*_GoStatement{goAssignmentStatement(
				target,
				"=",
				goCall(cloner, goRaw(source)),
			)}
		}
		resolved := resolveCloneType(type_, substitutions)
		if resolved == type_ {
			return nil
		}
		return b.cloneAssignStatementsWithSubstitutions(resolved, nil, source, target)
	}

	resolved := resolveCloneType(type_, substitutions)
	if cloneTypeUsesNullablePointer(resolved) {
		valueName := b.variableName("clonedValue")
		thenBlock := goBlock(
			goAssignmentStatement(valueName, ":=", goRaw("*"+source)),
		)
		thenBlock.append(b.cloneAssignStatementsWithSubstitutions(
			withoutCloneNullable(type_),
			substitutions,
			"(*"+source+")",
			valueName,
		)...)
		thenBlock.append(goAssignmentStatement(target, "=", goRaw("&"+valueName)))
		return []*_GoStatement{goIfStatement(nil, goRaw(source+" != nil"), thenBlock, nil)}
	}

	switch resolved.Kind {
	case model.TypeKindScalar:
		if resolved.Scalar == model.ScalarBinary {
			return b.cloneSliceAssignStatements(resolved, nil, nil, source, target)
		}
	case model.TypeKindList:
		elementType := resolved.List.Value
		if type_.Kind == model.TypeKindList {
			elementType = type_.List.Value
		}
		return b.cloneSliceAssignStatements(resolved, elementType, substitutions, source, target)
	case model.TypeKindMap:
		b.imports.add(&Import{Path: "maps"})
		keyName := b.variableName("key")
		itemName := b.variableName("item")
		clonedItemName := b.variableName("clonedItem")
		itemType := resolved.Map.Value
		if type_.Kind == model.TypeKindMap {
			itemType = type_.Map.Value
		}
		itemStatements := b.cloneAssignStatementsWithSubstitutions(
			itemType,
			substitutions,
			itemName,
			clonedItemName,
		)
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
		return []*_GoStatement{goAssignmentStatement(
			target,
			"=",
			b.cloneDataExpressionWithSubstitutions(type_, substitutions, source),
		)}
	}
	return nil
}

func (b *_CloneBuilder) cloneSliceAssignStatements(
	type_ *model.Type,
	elementType *model.Type,
	substitutions map[*model.TypeParameter]*model.Type,
	source string,
	target string,
) []*_GoStatement {
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
	elementStatements := b.cloneAssignStatementsWithSubstitutions(
		elementType,
		substitutions,
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
	return b.cloneDataExpressionWithSubstitutions(type_, nil, source)
}

func (b *_CloneBuilder) cloneDataExpressionWithSubstitutions(
	type_ *model.Type,
	substitutions map[*model.TypeParameter]*model.Type,
	source string,
) *_GoExpression {
	resolvedType := resolveCloneType(type_, substitutions)
	nonNullableType := withoutCloneNullable(resolvedType)
	castedType := castType(nonNullableType)
	b.imports.addMany(castedType.Imports)
	cloners := b.cloneTypeArgumentExpressions(type_, substitutions)
	if importedCloneType(nonNullableType) {
		return b.cloneImportedDataExpression(nonNullableType, source, cloners)
	}
	if len(nonNullableType.TypeArguments) == 0 {
		return goCall(source + ".Clone")
	}
	return goCall(source+".CloneBy", cloners...)
}

func (b *_CloneBuilder) cloneTypeArgumentExpressions(
	type_ *model.Type,
	substitutions map[*model.TypeParameter]*model.Type,
) []*_GoExpression {
	cloners := make([]*_GoExpression, 0, len(type_.TypeArguments))
	for _, argument := range type_.TypeArguments {
		if argument.Kind == model.TypeKindTypeParameter {
			if cloner := b.typeParameterCloner[argument.TypeParameter]; cloner != "" {
				cloners = append(cloners, goRaw(cloner))
				continue
			}
		}
		cloners = append(cloners, b.cloneFunctionExpression(resolveCloneType(argument, substitutions)))
	}
	return cloners
}

func importedCloneType(type_ *model.Type) bool {
	return type_.ExternalDomain != "" || type_.ExternalImportPath != ""
}

func (b *_CloneBuilder) cloneImportedDataExpression(
	type_ *model.Type,
	source string,
	cloners []*_GoExpression,
) *_GoExpression {
	castedType := castType(type_)
	plain := castedType.Plain
	key := _CloneDataKey{data: type_.Data, typePlain: plain}
	if activeClone := b.activeImportedClone[key]; activeClone != "" {
		return goCall(activeClone, goRaw(source))
	}
	if len(type_.TypeArguments) == 0 {
		return b.cloneImportedDataByMarshaling(type_, source, plain)
	}
	return b.cloneImportedGenericData(type_, source, plain, key, cloners)
}

func (b *_CloneBuilder) cloneImportedDataByMarshaling(
	type_ *model.Type,
	source string,
	plain string,
) *_GoExpression {
	clonerName := b.variableName("cloner")
	okName := b.variableName("clonerOK")
	body := goBlock(goIfStatement(
		goAssignments(
			[]string{clonerName, okName},
			":=",
			goRaw(fmt.Sprintf("any(value).(interface { Clone() %s })", plain)),
		),
		goRaw(okName),
		goBlock(goReturnStatement(goCall(clonerName+".Clone"))),
		nil,
	))
	body.append(goReturnStatement(b.cloneImportedDataMarshaledExpression(type_, "value", plain)))
	return goCallExpression(
		goFunctionExpression(goFunction(
			[]*_GoParameter{goParameter("value", plain)},
			plain,
			body,
		)),
		goRaw(source),
	)
}

func (b *_CloneBuilder) cloneImportedDataMarshaledExpression(
	type_ *model.Type,
	source string,
	plain string,
) *_GoExpression {
	b.imports.add(&Import{Path: "go.yorun.ai/vine/util/vcode"})
	if type_.ContainsBinaryType() {
		return goRaw(fmt.Sprintf(
			"*vcode.MustUnmarshalCbor[%s](vcode.MustMarshalCbor(%s))",
			plain,
			source,
		))
	}
	return goCall(
		"vcode.MustUnmarshalJson["+plain+"]",
		goCall("vcode.MustMarshalJson", goRaw(source)),
	)
}

func (b *_CloneBuilder) cloneImportedGenericData(
	type_ *model.Type,
	source string,
	plain string,
	key _CloneDataKey,
	cloners []*_GoExpression,
) *_GoExpression {
	cloneFunctionName := b.variableName("cloneImportedData")
	callbackParameters := make([]*_GoParameter, 0, len(type_.TypeArguments))
	callbackNames := make([]string, 0, len(type_.TypeArguments))
	callbackTypes := make([]string, 0, len(type_.TypeArguments))
	for _, argument := range type_.TypeArguments {
		castedArgument := castType(argument)
		b.imports.addMany(castedArgument.Imports)
		callbackName := b.variableName("cloneImportedArgument")
		callbackType := fmt.Sprintf("func(%s) %s", castedArgument.Plain, castedArgument.Plain)
		callbackParameters = append(callbackParameters, goParameter(callbackName, callbackType))
		callbackNames = append(callbackNames, callbackName)
		callbackTypes = append(callbackTypes, callbackType)
	}

	clonerName := b.variableName("cloner")
	okName := b.variableName("clonerOK")
	cloneBody := goBlock(goIfStatement(
		goAssignments(
			[]string{clonerName, okName},
			":=",
			goRaw(fmt.Sprintf(
				"any(value).(interface { CloneBy(%s) %s })",
				strings.Join(callbackTypes, ", "),
				plain,
			)),
		),
		goRaw(okName),
		goBlock(goReturnStatement(goCall(
			clonerName+".CloneBy",
			goRawExpressions(callbackNames)...,
		))),
		nil,
	))
	if importedCloneSchemaSupported(type_) {
		cloneBody.append(goAssignmentStatement("cloned", ":=", goRaw("value")))
	} else {
		cloneBody.append(goAssignmentStatement(
			"cloned",
			":=",
			b.cloneImportedDataMarshaledExpression(type_, "value", plain),
		))
	}

	b.activeImportedClone[key] = cloneFunctionName
	savedCloners := make(map[*model.TypeParameter]string, len(type_.Data.TypeParameters))
	hadCloner := make(map[*model.TypeParameter]bool, len(type_.Data.TypeParameters))
	for index, parameter := range type_.Data.TypeParameters {
		savedCloners[parameter], hadCloner[parameter] = b.typeParameterCloner[parameter]
		if index < len(callbackNames) {
			b.typeParameterCloner[parameter] = callbackNames[index]
		}
	}
	substitutions := cloneDataSubstitutions(type_)
	for _, member := range type_.Data.Members {
		memberType := contextualizeImportedCloneType(member.Type, type_)
		if !importedCloneMemberTypeSupported(memberType) {
			continue
		}
		memberName := nameutil.ToCamel(member.Name)
		cloneBody.append(b.cloneAssignStatementsWithSubstitutions(
			memberType,
			substitutions,
			"value."+memberName,
			"cloned."+memberName,
		)...)
	}
	for _, parameter := range type_.Data.TypeParameters {
		if hadCloner[parameter] {
			b.typeParameterCloner[parameter] = savedCloners[parameter]
		} else {
			delete(b.typeParameterCloner, parameter)
		}
	}
	delete(b.activeImportedClone, key)
	cloneBody.append(goReturnStatement(goRaw("cloned")))

	outerParameters := append([]*_GoParameter{goParameter("value", plain)}, callbackParameters...)
	outerArguments := append([]*_GoExpression{goRaw(source)}, cloners...)
	outerBody := goBlock(
		goVariableStatement(cloneFunctionName, fmt.Sprintf("func(%s) %s", plain, plain)),
		goAssignmentStatement(
			cloneFunctionName,
			"=",
			goFunctionExpression(goFunction(
				[]*_GoParameter{goParameter("value", plain)},
				plain,
				cloneBody,
			)),
		),
		goReturnStatement(goCall(cloneFunctionName, goRaw("value"))),
	)
	return goCallExpression(
		goFunctionExpression(goFunction(outerParameters, plain, outerBody)),
		outerArguments...,
	)
}

func importedCloneSchemaSupported(importedType *model.Type) bool {
	for _, member := range importedType.Data.Members {
		if !importedCloneMemberTypeSupported(contextualizeImportedCloneType(member.Type, importedType)) {
			return false
		}
	}
	return true
}

func importedCloneMemberTypeSupported(type_ *model.Type) bool {
	if type_ == nil {
		return false
	}
	switch type_.Kind {
	case model.TypeKindList:
		return type_.List != nil && importedCloneMemberTypeSupported(type_.List.Value)
	case model.TypeKindMap:
		return type_.Map != nil &&
			importedCloneMemberTypeSupported(type_.Map.Key) &&
			importedCloneMemberTypeSupported(type_.Map.Value)
	case model.TypeKindData, model.TypeKindEnum:
		if type_.ExternalDomain != "" && type_.ExternalImportPath == "" {
			return false
		}
		for _, argument := range type_.TypeArguments {
			if !importedCloneMemberTypeSupported(argument) {
				return false
			}
		}
	}
	return true
}

func contextualizeImportedCloneType(type_ *model.Type, importedType *model.Type) *model.Type {
	if type_ == nil {
		return nil
	}
	contextualized := *type_
	switch type_.Kind {
	case model.TypeKindList:
		contextualized.List = &model.ListType{
			Value: contextualizeImportedCloneType(type_.List.Value, importedType),
		}
	case model.TypeKindMap:
		contextualized.Map = &model.MapType{
			Key:   contextualizeImportedCloneType(type_.Map.Key, importedType),
			Value: contextualizeImportedCloneType(type_.Map.Value, importedType),
		}
	case model.TypeKindData:
		contextualized.TypeArguments = make([]*model.Type, 0, len(type_.TypeArguments))
		for _, argument := range type_.TypeArguments {
			contextualized.TypeArguments = append(
				contextualized.TypeArguments,
				contextualizeImportedCloneType(argument, importedType),
			)
		}
		if type_.Data != nil && type_.Data.Domain == importedType.Data.Domain && type_.ExternalDomain == "" {
			contextualized.ExternalDomain = importedType.ExternalDomain
			contextualized.ExternalAlias = importedType.ExternalAlias
			contextualized.ExternalAliasExplicit = importedType.ExternalAliasExplicit
			contextualized.ExternalImportPath = importedType.ExternalImportPath
		}
	case model.TypeKindEnum:
		if type_.Enum != nil && type_.Enum.Domain == importedType.Data.Domain && type_.ExternalDomain == "" {
			contextualized.ExternalDomain = importedType.ExternalDomain
			contextualized.ExternalAlias = importedType.ExternalAlias
			contextualized.ExternalAliasExplicit = importedType.ExternalAliasExplicit
			contextualized.ExternalImportPath = importedType.ExternalImportPath
		}
	}
	return &contextualized
}

func goRawExpressions(values []string) []*_GoExpression {
	expressions := make([]*_GoExpression, 0, len(values))
	for _, value := range values {
		expressions = append(expressions, goRaw(value))
	}
	return expressions
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
