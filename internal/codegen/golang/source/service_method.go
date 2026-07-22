package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
	"strings"
)

type ServiceMethod struct {
	Name                        string
	SkelName                    string
	SpecName                    string
	CommentLines                []string
	Arguments                   []*MethodArgument
	ArgumentsData               *Data
	ValidateArguments           string
	ResultType                  *Type
	ValidateResult              string
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
			checkutil.Check(ok, "argument member %s not found", arg.SkelName)
			arg.MemberName = member.Name
		}
	}
	method.ValidateArguments = buildMethodValidateArguments(method)
	method.CommentLines = goMethodDocLines(method.Name, pm.Description, pm.Example, method.Arguments, method.ResultType, pm.OutputDescription, pm.OutputExample)
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

func buildMethodValidateResult(method *model.Method, resultType *Type) string {
	if method.ResultType == nil || resultType == nil || !typeNeedsCheck(method.ResultType, map[*model.Data]bool{}) {
		return ""
	}

	lines := []string{"func(value any) error {"}
	if method.ResultType.Nullable {
		lines = append(lines, "\tif value == nil {", "\t\treturn nil", "\t}")
	}
	lines = append(lines, fmt.Sprintf("\tret := value.(%s)", resultType.Plain))
	lines = append(lines, buildTypeCheckLines(method.ResultType, "ret", `"result"`, "\t", 0)...)
	lines = append(lines, "\treturn nil", "}")
	return strings.Join(lines, "\n")
}

func buildMethodValidateArguments(method *ServiceMethod) string {
	if method.ArgumentsData == nil {
		return ""
	}

	needsCheck := false
	for _, argument := range method.Arguments {
		if typeNeedsCheck(argument.ParsedType, map[*model.Data]bool{}) {
			needsCheck = true
			break
		}
	}
	if !needsCheck {
		return ""
	}

	lines := []string{"func(value any) error {"}
	lines = append(lines, fmt.Sprintf("\targs := value.(*%s)", method.ArgumentsData.Name))
	for _, argument := range method.Arguments {
		if !typeNeedsCheck(argument.ParsedType, map[*model.Data]bool{}) {
			continue
		}
		lines = append(lines, buildTypeCheckLines(argument.ParsedType, "args."+argument.MemberName, fmt.Sprintf("rpc.JoinPath(%q, %q)", "arguments", argument.MemberName), "\t", 0)...)
	}
	lines = append(lines, "\treturn nil", "}")
	return strings.Join(lines, "\n")
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
	return &MethodArgument{
		Name:        nameutil.ToLowerCamel(p.Name),
		SkelName:    p.Name,
		Description: common.MergeDescriptionAndExample(p.Description, p.Example),
		Type:        argType,
		ParsedType:  p.Type,
	}
}
