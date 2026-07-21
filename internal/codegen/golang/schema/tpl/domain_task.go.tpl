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
					{{- if $trigger.Example }}
					Example: {{ quote $trigger.Example }},
					{{- end }}
					{{- if $trigger.InputDescription }}
					InputDescription: {{ quote $trigger.InputDescription }},
					{{- end }}
					{{- if $trigger.Arguments }}
					Arguments: []*skel.MemberSchema{
						{{- range $argument := $trigger.Arguments }}
						{
							Name: {{ quote $argument.Name }},
							{{- if $argument.Description }}
							Description: {{ quote $argument.Description }},
							{{- end }}
							{{- if $argument.Example }}
							Example: {{ quote $argument.Example }},
							{{- end }}
							Type: {{ template "typeSchema" $argument.Type }},
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
	},
	{{- end }}
{{- end }}
