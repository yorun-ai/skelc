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
			{{- template "deprecatedFields" $service }}
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
				{{ template "methodSchemaValue" $method }},
				{{- end }}
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
