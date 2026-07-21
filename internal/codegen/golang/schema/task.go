package schema

import "go.yorun.ai/skelc/model"

func (g *_Gen) buildTaskSchema(task *model.Task) *_TaskSchema {
	schema := &_TaskSchema{
		Name: task.Name, SkelName: task.SkelName, Hash: task.Hash,
		Description: task.Description, Triggers: make([]*_TriggerSchema, 0, len(task.Triggers)),
	}
	for _, trigger := range task.Triggers {
		schema.Triggers = append(schema.Triggers, &_TriggerSchema{
			Name: trigger.Name, SkelName: trigger.SkelName, Hash: trigger.Hash,
			Description: trigger.Description, Example: trigger.Example,
			InputDescription: trigger.InputDescription,
			Arguments:        g.buildArgumentSchemas(trigger.Arguments),
		})
	}
	return schema
}
