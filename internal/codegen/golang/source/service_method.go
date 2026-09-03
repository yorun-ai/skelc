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
	CloneArguments              *_GoFunction
	ResultType                  *Type
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

type MethodArgument struct {
	Name        string
	SkelName    string
	MemberName  string
	Description string
	Type        *Type
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
	}
}
