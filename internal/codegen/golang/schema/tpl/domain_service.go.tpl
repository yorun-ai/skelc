{{- define "domainServices" -}}
{{- if .Schema.Services }}
	Services: []*skel.ServiceSchema{
		{{- range $service := .Schema.Services }}
		{
			Name: {{ quote $service.Name }},
			SkelName: {{ quote $service.SkelName }},
			{{- if $service.Description }}
			Description: {{ quote $service.Description }},
			{{- end }}
			Hash: {{ quote $service.Hash }},
			Pub: {{ $service.Pub }},
			AuthMode: {{ authLiteral $service.AuthMode }},
			{{- if $service.Audiences }}
			Audiences: []*skel.ActorAudienceSchema{
				{{- range $actor := $service.Audiences }}
				{Name: {{ quote $actor.Name }}, SkelName: {{ quote $actor.SkelName }}{{ with $actor.Via }}, Via: {{ viaLiteral . }}{{ end }}},
				{{- end }}
			},
			{{- end }}
			{{- if $service.Require }}
			Require: {{ template "permissionRequire" $service.Require }},
			{{- end }}
			{{- if $service.Methods }}
			Methods: []*skel.MethodSchema{
				{{- range $method := $service.Methods }}
				{
					Name: {{ quote $method.Name }},
					SkelName: {{ quote $method.SkelName }},
					{{- if $method.Description }}
					Description: {{ quote $method.Description }},
					{{- end }}
					Hash: {{ quote $method.Hash }},
					{{- if $method.Example }}
					Example: {{ quote $method.Example }},
					{{- end }}
					AuthMode: {{ authLiteral $method.AuthMode }},
					{{- if $method.Require }}
					Require: {{ template "permissionRequire" $method.Require }},
					{{- end }}
					{{- if $method.InputDescription }}
					InputDescription: {{ quote $method.InputDescription }},
					{{- end }}
					{{- if $method.OutputDescription }}
					OutputDescription: {{ quote $method.OutputDescription }},
					{{- end }}
					{{- if $method.OutputExample }}
					OutputExample: {{ quote $method.OutputExample }},
					{{- end }}
					{{- if $method.Arguments }}
					Arguments: []*skel.MemberSchema{
						{{- range $argument := $method.Arguments }}
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
					{{- if $method.ResultType }}
					ResultType: {{ template "typeSchema" $method.ResultType }},
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
