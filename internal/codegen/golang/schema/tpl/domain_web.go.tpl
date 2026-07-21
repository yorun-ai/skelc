{{- define "domainWebs" -}}
{{- if .Schema.Webs }}
	Webs: []*skel.WebSchema{
		{{- range $web := .Schema.Webs }}
		{
			Name: {{ quote $web.Name }},
			SkelName: {{ quote $web.SkelName }},
			{{- if $web.Description }}
			Description: {{ quote $web.Description }},
			{{- end }}
			Hash: {{ quote $web.Hash }},
			{{- if $web.Audiences }}
			Audiences: []*skel.ActorAudienceSchema{
				{{- range $actor := $web.Audiences }}
				{Name: {{ quote $actor.Name }}, SkelName: {{ quote $actor.SkelName }}{{ with $actor.Via }}, Via: {{ viaLiteral . }}{{ end }}},
				{{- end }}
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
