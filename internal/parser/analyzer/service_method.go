package analyzer

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
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

func parseMethods(reporter *diagnosticReporter, owner *grammar.Identifier, methods []*grammar.Method) ([]*model.Method, bool) {
	parsedMethods := make([]*model.Method, 0, len(methods))
	methodPos := map[string]lexer.Position{}
	valid := true

	for _, grammarMethod := range methods {
		method, methodValid := parseMethod(reporter, grammarMethod)
		valid = methodValid && valid
		duplicatedPosition, duplicated := methodPos[method.Name]
		if duplicated {
			reporter.reportf("%s duplicated method %s found, also present at %s", method.Pos, method.Name, duplicatedPosition)
			valid = false
			continue
		}
		if method.ArgumentsData != nil {
			method.ArgumentsData.Name = fmt.Sprintf("%s%s", owner.Value, method.ArgumentsData.Name)
		}
		methodPos[method.Name] = lexer.Position{Filename: method.Pos.File, Line: method.Pos.Line, Column: method.Pos.Column}
		parsedMethods = append(parsedMethods, method)
	}

	valid = reporter.check(len(parsedMethods) > 0, "%s missing method for %s", owner.Pos, owner.Value) && valid
	return parsedMethods, valid
}

func parseMethod(reporter *diagnosticReporter, gm *grammar.Method) (*model.Method, bool) {
	valid := checkCase(reporter, "Method", caseTypeLowerCamel, gm.Name)
	meta, metaValid := parseDecoratorMeta(reporter, gm.Decorators, decoratorContext{
		allowDesc: true,
	})
	valid = metaValid && valid
	require, requireValid := parseRequire(reporter, gm.Require)
	valid = requireValid && valid
	authMode, authModeValid := parseAuthMode(reporter, methodAuthMarker(gm), model.AuthModeUnset)
	valid = authModeValid && valid
	method := &model.Method{
		Pos:         position(gm.Name.Pos),
		Name:        gm.Name.Value,
		SkelName:    gm.Name.Value,
		Description: meta.Description,
		Auth:        authMode,
		Require:     require,
		Arguments:   []*model.Argument{},
	}
	input := methodInput(gm)
	output := methodOutput(gm)

	if input != nil {
		inputMeta, inputValid := parseDecoratorMeta(reporter, input.Decorators, decoratorContext{
			allowDesc: true,
		})
		valid = inputValid && valid
		method.InputDescription = inputMeta.Description
		argPos := map[string]lexer.Position{}
		for _, grammarArgument := range input.Arguments {
			arg, argumentValid := parseArgument(reporter, grammarArgument)
			valid = argumentValid && valid
			duplicatedPosition, duplicated := argPos[arg.Name]
			if duplicated {
				reporter.reportf("%s duplicated Argument %s found, also present at %s", arg.Pos, arg.Name, duplicatedPosition)
				valid = false
				continue
			}
			argPos[arg.Name] = lexer.Position{Filename: arg.Pos.File, Line: arg.Pos.Line, Column: arg.Pos.Column}
			method.Arguments = append(method.Arguments, arg)
		}
	}
	if output != nil {
		outputMeta, outputValid := parseAnnotations(reporter, output.Decorators)
		valid = outputValid && valid
		method.OutputDescription = outputMeta.Description
		if outputMeta.HasExample {
			method.OutputExample = outputMeta.Example
		}
		method.ResultType, outputValid = parseType(reporter, output.Type)
		valid = outputValid && valid
	}

	if len(method.Arguments) > 0 {
		method.ArgumentsData = &model.Data{
			Name:    fmt.Sprintf("%sArguments", nameutil.ToCamel(method.Name)),
			Members: buildArgumentMembers(method.Arguments),
		}
	}

	return method, valid
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

func parseArgument(reporter *diagnosticReporter, ga *grammar.Argument) (*model.Argument, bool) {
	valid := checkCase(reporter, "Argument", caseTypeLowerCamel, ga.Name)
	meta, metaValid := parseAnnotations(reporter, ga.Decorators)
	valid = metaValid && valid
	argType, typeValid := parseType(reporter, ga.Type)
	valid = typeValid && valid
	arg := &model.Argument{
		Pos:         position(ga.Name.Pos),
		Name:        ga.Name.Value,
		Description: meta.Description,
		Type:        argType,
	}
	if meta.HasExample {
		arg.Example = meta.Example
	}
	return arg, valid
}
