{{- define "domainTasks" -}}
{{- if .Schema.Tasks }}
	Tasks: []*skel.TaskSchema{
		{{- range $task := .Schema.Tasks }}
		{
			Name: {{ quote $task.Name }},
			SkelName: {{ quote $task.SkelName }},
			{{- if $task.Description }}
			Description: {{ quote $task.Description }},
			{{- end }}
			Hash: {{ quote $task.Hash }},
			{{- if $task.Triggers }}
			Triggers: []*skel.TriggerSchema{
				{{- range $trigger := $task.Triggers }}
				{
					Name: {{ quote $trigger.Name }},
					SkelName: {{ quote $trigger.SkelName }},
					{{- if $trigger.Description }}
					Description: {{ quote $trigger.Description }},
					{{- end }}
					Hash: {{ quote $trigger.Hash }},
					{{- if $trigger.InputDescription }}
					InputDescription: {{ quote $trigger.InputDescription }},
					{{- end }}
					{{- if $trigger.ArgumentsSensitive }}
					ArgumentsSensitive: true,
					{{- end }}
					{{- if $trigger.Arguments }}
					Arguments: []*skel.MemberSchema{
						{{- range $argument := $trigger.Arguments }}
						{{ template "memberSchema" $argument }},
						{{- end }}
					},
					{{- end }}
				},
				{{- end }}
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
