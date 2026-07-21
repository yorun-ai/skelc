{{- define "domainResources" -}}
{{- if .Schema.Resources }}
	Resources: []*skel.ResourceSchema{
		{{- range $resource := .Schema.Resources }}
		{
			Name: {{ quote $resource.Name }},
			SkelName: {{ quote $resource.SkelName }},
			{{- if $resource.Description }}
			Description: {{ quote $resource.Description }},
			{{- end }}
			Hash: {{ quote $resource.Hash }},
			{{- if $resource.Checks }}
			Checks: []*skel.ResourceCheckSchema{
				{{- range $check := $resource.Checks }}
				{
					Name: {{ quote $check.Name }},
					Method: {{ template "methodSchema" $check.Method }},
					{{- if $check.Arguments }}
					Arguments: []*skel.MemberSchema{
						{{- range $argument := $check.Arguments }}
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
			Actions: []*skel.ResourceActionSchema{
				{{- range $action := $resource.Actions }}
				{
					Name: {{ quote $action.Name }},
					PermissionCode: {{ quote $action.PermissionCode }},
					{{- if $action.Description }}
					Description: {{ quote $action.Description }},
					{{- end }}
					{{- if $action.Checks }}
					Checks: []*skel.ResourceCheckSchema{
						{{- range $check := $action.Checks }}
						{
							Name: {{ quote $check.Name }},
							Method: {{ template "methodSchema" $check.Method }},
							{{- if $check.Arguments }}
							Arguments: []*skel.MemberSchema{
								{{- range $argument := $check.Arguments }}
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
			{{- if $resource.CheckService }}
			CheckService: {{ template "serviceSchema" $resource.CheckService }},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
