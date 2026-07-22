package analyzer

import (
	"fmt"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

func parseData(gs *grammar.Data) *model.Data {
	return parseDataLike(gs, model.DataKindData)
}

func parseConfig(gs *grammar.Data) *model.Data {
	return parseDataLike(gs, model.DataKindConfig)
}

func parseEvent(ge *grammar.Event) *model.Data {
	members := []*grammar.DataMember{}
	if ge.Payload != nil {
		members = ge.Payload.Members
	}
	return parseDataLike(&grammar.Data{
		Pos:            ge.Pos,
		Decorators:     ge.Decorators,
		Pub:            ge.Pub,
		Name:           ge.Name,
		Qualifier:      ge.Qualifier,
		Members:        members,
		TypeParameters: ge.TypeParameters,
	}, model.DataKindEvent)
}

func parseDataLike(gs *grammar.Data, kind model.DataKind) *model.Data {
	checkCase("Data", caseTypeCamel, gs.Name)
	if kind == model.DataKindConfig {
		checkutil.Check(strings.HasSuffix(gs.Name.Value, "Config"), "%s Config name must end with Config", gs.Name.Pos)
		checkutil.CheckNotNil(
			gs.Qualifier,
			"%s Config %s requires lifecycle qualifier eternal/instant",
			gs.Name.Pos, gs.Name.Value,
		)
		checkutil.Check(
			gs.Qualifier.Value == string(model.ConfigLifecycleEternal) || gs.Qualifier.Value == string(model.ConfigLifecycleInstant),
			"%s Config %s has invalid lifecycle qualifier %s, expected eternal/instant",
			gs.Qualifier.Pos, gs.Name.Value, gs.Qualifier.Value,
		)
		checkutil.Check(len(gs.TypeParameters) == 0,
			"%s Config %s does not support type parameters",
			gs.Name.Pos, gs.Name.Value,
		)
	} else if kind == model.DataKindEvent {
		checkutil.Check(strings.HasSuffix(gs.Name.Value, "Event"), "%s Event name must end with Event", gs.Name.Pos)
		checkutil.Check(len(gs.TypeParameters) == 0,
			"%s Event %s does not support type parameters",
			gs.Name.Pos, gs.Name.Value,
		)
		checkutil.CheckFuncAt(gs.Name.Pos, gs.Qualifier == nil, func() string {
			return fmt.Sprintf("%s Event %s does not support qualifier %s", gs.Qualifier.Pos, gs.Name.Value, gs.Qualifier.Value)
		})
	} else {
		checkNotReservedKindSuffix("Data", gs.Name)
		checkutil.CheckFuncAt(gs.Name.Pos, gs.Qualifier == nil, func() string {
			return fmt.Sprintf("%s Data %s does not support qualifier %s", gs.Qualifier.Pos, gs.Name.Value, gs.Qualifier.Value)
		})
	}
	parsedData := &model.Data{
		Pos:     position(gs.Name.Pos),
		Name:    gs.Name.Value,
		Kind:    kind,
		Pub:     gs.Pub,
		Members: []*model.DataMember{},
	}
	if gs.Qualifier != nil {
		parsedData.Lifecycle = model.ConfigLifecycle(gs.Qualifier.Value)
	}
	meta := parseDecoratorMeta(gs.Decorators, decoratorContext{
		allowDesc: true,
	})
	parsedData.Description = meta.Description
	typeParamPos := map[string]lexer.Position{}
	memberPos := map[string]lexer.Position{}

	for _, grammarTypeParameter := range gs.TypeParameters {
		typeParameter := parseTypeParameter(grammarTypeParameter)
		duplicatedPosition, duplicated := typeParamPos[typeParameter.Name]
		checkutil.CheckFuncAt(typeParameter.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated TypeParameter %s found, also present at %s",
				typeParameter.Pos, typeParameter.Name, duplicatedPosition)
		})
		typeParamPos[typeParameter.Name] = lexer.Position{Filename: typeParameter.Pos.File, Line: typeParameter.Pos.Line, Column: typeParameter.Pos.Column}
		parsedData.TypeParameters = append(parsedData.TypeParameters, typeParameter)
	}

	for _, grammarMember := range gs.Members {
		member := parseDataMember(grammarMember)
		duplicatedPosition, duplicated := memberPos[member.Name]
		checkutil.CheckFuncAt(member.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated DataMember %s found, also present at %s",
				member.Pos, member.Name, duplicatedPosition)
		})
		memberPos[member.Name] = lexer.Position{Filename: member.Pos.File, Line: member.Pos.Line, Column: member.Pos.Column}
		parsedData.Members = append(parsedData.Members, member)
	}

	return parsedData
}

func parseTypeParameter(gtp *grammar.TypeParameter) *model.TypeParameter {
	checkCaseAdvanced("TypeParameter", "T", "", caseTypeCamel, gtp.Name)
	checkutil.CheckNot(gtp.Nullable, "%s TypeParameter %s cannot be nullable", gtp.Pos, gtp.Name.Value)
	return &model.TypeParameter{
		Name: gtp.Name.Value,
		Pos:  position(gtp.Name.Pos),
	}
}

func parseDataMember(gsm *grammar.DataMember) *model.DataMember {
	checkCase("DataMember", caseTypeLowerCamel, gsm.Name)
	meta := parseDecoratorMeta(gsm.Decorators, decoratorContext{
		allowDesc:    true,
		allowExample: true,
		requireDesc:  true,
	})
	memberType := parseType(gsm.Type)
	return &model.DataMember{
		Pos:         position(gsm.Name.Pos),
		Name:        gsm.Name.Value,
		Description: meta.Description,
		Example:     meta.Example,
		Type:        memberType,
	}
}
