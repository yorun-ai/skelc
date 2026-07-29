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
			{{ template "deprecatedFields" $resource }}
			Hash: {{ quote $resource.Hash }},
			{{- if $resource.Checks }}
			Checks: []*skel.ResourceCheckSchema{
				{{- range $check := $resource.Checks }}
				{
					Name: {{ quote $check.Name }},
					{{ template "deprecatedFields" $check }}
					Method: {{ template "methodSchema" $check.Method }},
					{{- if $check.Arguments }}
					Arguments: []*skel.MemberSchema{
						{{- range $argument := $check.Arguments }}
						{{ template "memberSchema" $argument }},
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
					{{ template "deprecatedFields" $action }}
					{{- if $action.Checks }}
					Checks: []*skel.ResourceCheckSchema{
						{{- range $check := $action.Checks }}
						{
							Name: {{ quote $check.Name }},
							{{ template "deprecatedFields" $check }}
							Method: {{ template "methodSchema" $check.Method }},
							{{- if $check.Arguments }}
							Arguments: []*skel.MemberSchema{
								{{- range $argument := $check.Arguments }}
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
			{{- if $resource.CheckService }}
			CheckService: {{ template "serviceSchema" $resource.CheckService }},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
