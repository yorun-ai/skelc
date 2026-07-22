package analyzer

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/model"
)

func parseTask(gt *grammar.Task) *model.Task {
	checkCaseAdvanced("Task", "", "Task", caseTypeCamel, gt.Name)
	meta := parseDecoratorMeta(gt.Decorators, decoratorContext{
		allowDesc: true,
	})
	checkutil.CheckNot(meta.HasExample, "%s task does not support decorator @example", gt.Name.Pos)
	triggers := parseTaskTriggers(gt.Name, gt.Triggers)
	return &model.Task{
		Pos:         position(gt.Name.Pos),
		Name:        gt.Name.Value,
		Description: meta.Description,
		Triggers:    triggers,
	}
}

func parseTaskTriggers(owner *grammar.Identifier, triggers []*grammar.TaskTrigger) []*model.TaskTrigger {
	parsedTriggers := make([]*model.TaskTrigger, 0, len(triggers))
	triggerPos := map[string]lexer.Position{}

	for _, grammarTrigger := range triggers {
		trigger := parseTaskTrigger(grammarTrigger)
		duplicatedPosition, duplicated := triggerPos[trigger.Name]
		checkutil.CheckFuncAt(trigger.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated task trigger %s found, also present at %s",
				trigger.Pos, trigger.Name, duplicatedPosition)
		})
		if trigger.ArgumentsData != nil {
			trigger.ArgumentsData.Name = fmt.Sprintf("%s%s", owner.Value, trigger.ArgumentsData.Name)
		}
		triggerPos[trigger.Name] = lexer.Position{Filename: trigger.Pos.File, Line: trigger.Pos.Line, Column: trigger.Pos.Column}
		parsedTriggers = append(parsedTriggers, trigger)
	}

	checkutil.Check(len(parsedTriggers) > 0, "%s missing task trigger for %s", owner.Pos, owner.Value)

	return parsedTriggers
}

func parseTaskTrigger(gt *grammar.TaskTrigger) *model.TaskTrigger {
	checkCase("TaskTrigger", caseTypeLowerCamel, gt.Name)
	meta := parseDecoratorMeta(gt.Decorators, decoratorContext{
		allowDesc: true,
	})

	trigger := &model.TaskTrigger{
		Pos:         position(gt.Name.Pos),
		Name:        gt.Name.Value,
		SkelName:    gt.Name.Value,
		Description: meta.Description,
		Arguments:   []*model.Argument{},
	}
	if gt.Input == nil {
		return trigger
	}

	inputMeta := parseDecoratorMeta(gt.Input.Decorators, decoratorContext{
		allowDesc: true,
	})
	trigger.InputDescription = inputMeta.Description
	argPos := map[string]lexer.Position{}
	for _, grammarArgument := range gt.Input.Arguments {
		arg := parseArgument(grammarArgument)
		duplicatedPosition, duplicated := argPos[arg.Name]
		checkutil.CheckFuncAt(arg.Pos, !duplicated, func() string {
			return fmt.Sprintf("%s duplicated Argument %s found, also present at %s",
				arg.Pos, arg.Name, duplicatedPosition)
		})
		argPos[arg.Name] = lexer.Position{Filename: arg.Pos.File, Line: arg.Pos.Line, Column: arg.Pos.Column}
		trigger.Arguments = append(trigger.Arguments, arg)
	}

	if len(trigger.Arguments) > 0 {
		trigger.ArgumentsData = &model.Data{
			Name:    fmt.Sprintf("%sArguments", nameutil.ToCamel(trigger.Name)),
			Members: buildArgumentMembers(trigger.Arguments),
		}
	}

	return trigger
}
