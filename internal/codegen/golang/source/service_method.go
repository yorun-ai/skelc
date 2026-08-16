package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/model"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
)

type ServiceMethod struct {
	Name                        string
	SkelName                    string
	SpecName                    string
	CommentLines                []string
	Arguments                   []*MethodArgument
	ArgumentsData               *Data
	ValidateArguments           *_GoFunction
	CloneArguments              *_GoFunction
	ResultType                  *Type
	ValidateResult              *_GoFunction
	CloneResult                 *_GoFunction
	CloneImports                []*Import
	ArgumentsSensitive          bool
	ResultSensitive             bool
	ArgumentsContainsBinaryType bool
	ResultContainsBinaryType    bool
}

func castServiceMethod(ps *model.Service, pm *model.Method) *ServiceMethod {
	methodArgs := make([]*MethodArgument, 0, len(pm.Arguments))
	for _, argument := range pm.Arguments {
		castedArgument := castMethodArgument(argument)
		methodArgs = append(methodArgs, castedArgument)
	}
	resultType := castType(pm.ResultType)
	method := &ServiceMethod{
		Name:                        nameutil.ToCamel(pm.Name),
		SkelName:                    pm.Name,
		Arguments:                   methodArgs,
		ResultType:                  resultType,
		ValidateResult:              buildMethodValidateResult(pm, resultType),
		ArgumentsSensitive:          pm.ArgumentsSensitive,
		ResultSensitive:             pm.ResultSensitive,
		ArgumentsContainsBinaryType: methodArgumentsContainBinaryType(pm),
		ResultContainsBinaryType:    methodResultContainsBinaryType(pm),
	}
	method.SpecName = fmt.Sprintf("_%s%sSpec", ps.Name, method.Name)
	if pm.ArgumentsData != nil {
		method.ArgumentsData = castData(pm.ArgumentsData)
		method.ArgumentsData.Name = fmt.Sprintf("_%s", method.ArgumentsData.Name)
		for _, arg := range method.Arguments {
			member, ok := sliceutil.Find(method.ArgumentsData.Members, func(mem *DataMember) bool {
				return mem.SkelName == arg.SkelName
			})
			if ok {
				arg.MemberName = member.Name
			}
		}
	}
	method.ValidateArguments = buildMethodValidateArguments(method)
	buildMethodClones(pm, method)
	method.CommentLines = goMethodDocLines(
		method.Name,
		pm.Description,
		pm.Example,
		method.Arguments,
		method.ResultType,
		pm.OutputDescription,
		pm.OutputExample,
		pm.DeprecatedReason,
	)
	return method
}

func methodArgumentsContainBinaryType(method *model.Method) bool {
	for _, argument := range method.Arguments {
		if argument.Type.ContainsBinaryType() {
			return true
		}
	}
	return false
}

func methodResultContainsBinaryType(method *model.Method) bool {
	return method.ResultType.ContainsBinaryType()
}

func buildMethodValidateResult(method *model.Method, resultType *Type) *_GoFunction {
	if method.ResultType == nil || resultType == nil || !typeNeedsCheck(method.ResultType, map[*model.Data]bool{}) {
		return nil
	}

	body := goBlock()
	if method.ResultType.Nullable {
		body.append(goIfStatement(
			nil,
			goRaw("value == nil"),
			goBlock(goReturnStatement(goRaw("nil"))),
			nil,
		))
	}
	body.append(goAssignmentStatement("ret", ":=", goRaw(fmt.Sprintf("value.(%s)", resultType.Plain))))
	body.append(buildTypeCheckStatements(method.ResultType, "ret", `"result"`, 0)...)
	body.append(goReturnStatement(goRaw("nil")))
	return goFunction([]*_GoParameter{goParameter("value", "any")}, "error", body)
}

func buildMethodValidateArguments(method *ServiceMethod) *_GoFunction {
	if method.ArgumentsData == nil {
		return nil
	}

	needsCheck := false
	for _, argument := range method.Arguments {
		if typeNeedsCheck(argument.ParsedType, map[*model.Data]bool{}) {
			needsCheck = true
			break
		}
	}
	if !needsCheck {
		return nil
	}

	body := goBlock(
		goAssignmentStatement("args", ":=", goRaw(fmt.Sprintf("value.(*%s)", method.ArgumentsData.Name))),
	)
	for _, argument := range method.Arguments {
		if !typeNeedsCheck(argument.ParsedType, map[*model.Data]bool{}) {
			continue
		}
		body.append(buildTypeCheckStatements(
			argument.ParsedType,
			"args."+argument.MemberName,
			fmt.Sprintf("rpc.JoinPath(%q, %q)", "arguments", argument.MemberName),
			0,
		)...)
	}
	body.append(goReturnStatement(goRaw("nil")))
	return goFunction([]*_GoParameter{goParameter("value", "any")}, "error", body)
}

type MethodArgument struct {
	Name        string
	SkelName    string
	MemberName  string
	Description string
	Type        *Type
	ParsedType  *model.Type
}

func castMethodArgument(p *model.Argument) *MethodArgument {
	argType := castType(p.Type)
	description := common.MergeDescriptionAndExample(p.Description, p.Example)
	if p.Deprecated {
		if description != "" {
			description += " "
		}
		description += "Deprecated: " + common.EnsureSentence(p.DeprecatedReason)
	}
	return &MethodArgument{
		Name:        nameutil.ToLowerCamel(p.Name),
		SkelName:    p.Name,
		Description: description,
		Type:        argType,
		ParsedType:  p.Type,
	}
}
