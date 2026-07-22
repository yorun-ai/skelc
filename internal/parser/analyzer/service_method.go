package analyzer

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

func buildArgumentMembers(args []*model.Argument) []*model.DataMember {
	return sliceutil.Map(args, func(arg *model.Argument) *model.DataMember {
		return &model.DataMember{
			Name:        arg.Name,
			Description: arg.Description,
			Example:     arg.Example,
			Type:        arg.Type,
		}
	})
}

func parseMethods(owner *grammar.Identifier, methods []*grammar.Method) []*model.Method {
	parsedMethods := make([]*model.Method, 0, len(methods))
	methodPos := map[string]lexer.Position{}

	for _, grammarMethod := range methods {
		method := parseMethod(grammarMethod)
		duplicatedPosition, duplicated := methodPos[method.Name]
		checkutil.CheckFuncAt(method.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated method %s found, also present at %s", method.Pos, method.Name, duplicatedPosition)
		})
		if method.ArgumentsData != nil {
			method.ArgumentsData.Name = fmt.Sprintf("%s%s", owner.Value, method.ArgumentsData.Name)
		}
		methodPos[method.Name] = lexer.Position{Filename: method.Pos.File, Line: method.Pos.Line, Column: method.Pos.Column}
		parsedMethods = append(parsedMethods, method)
	}

	checkutil.Check(len(parsedMethods) > 0, "%s missing method for %s", owner.Pos, owner.Value)
	return parsedMethods
}

func parseMethod(gm *grammar.Method) *model.Method {
	checkCase("Method", caseTypeLowerCamel, gm.Name)
	meta := parseDecoratorMeta(gm.Decorators, decoratorContext{
		allowDesc: true,
	})
	method := &model.Method{
		Pos:         position(gm.Name.Pos),
		Name:        gm.Name.Value,
		SkelName:    gm.Name.Value,
		Description: meta.Description,
		Auth:        parseAuthMode(methodAuthMarker(gm), model.AuthModeUnset),
		Require:     parseRequire(gm.Require),
		Arguments:   []*model.Argument{},
	}
	input := methodInput(gm)
	output := methodOutput(gm)

	if input != nil {
		inputMeta := parseDecoratorMeta(input.Decorators, decoratorContext{
			allowDesc: true,
		})
		method.InputDescription = inputMeta.Description
		argPos := map[string]lexer.Position{}
		for _, grammarArgument := range input.Arguments {
			arg := parseArgument(grammarArgument)
			duplicatedPosition, duplicated := argPos[arg.Name]
			checkutil.CheckFuncAt(arg.Pos, !duplicated, func() string {
				return fmt.Sprintf("%s duplicated Argument %s found, also present at %s", arg.Pos, arg.Name, duplicatedPosition)
			})
			argPos[arg.Name] = lexer.Position{Filename: arg.Pos.File, Line: arg.Pos.Line, Column: arg.Pos.Column}
			method.Arguments = append(method.Arguments, arg)
		}
	}
	if output != nil {
		outputMeta := parseAnnotations(output.Decorators)
		method.OutputDescription = outputMeta.Description
		if outputMeta.HasExample {
			method.OutputExample = outputMeta.Example
		}
		method.ResultType = parseType(output.Type)
	}

	if len(method.Arguments) > 0 {
		method.ArgumentsData = &model.Data{
			Name:    fmt.Sprintf("%sArguments", nameutil.ToCamel(method.Name)),
			Members: buildArgumentMembers(method.Arguments),
		}
	}

	return method
}

func methodAuthMarker(gm *grammar.Method) *grammar.AuthMarker {
	return gm.Auth
}

func methodInput(gm *grammar.Method) *grammar.MethodInput {
	return gm.Input
}

func methodOutput(gm *grammar.Method) *grammar.MethodOutput {
	return gm.Output
}

func parseArgument(ga *grammar.Argument) *model.Argument {
	checkCase("Argument", caseTypeLowerCamel, ga.Name)
	meta := parseAnnotations(ga.Decorators)
	argType := parseType(ga.Type)
	arg := &model.Argument{
		Pos:         position(ga.Name.Pos),
		Name:        ga.Name.Value,
		Description: meta.Description,
		Type:        argType,
	}
	if meta.HasExample {
		arg.Example = meta.Example
	}
	return arg
}
