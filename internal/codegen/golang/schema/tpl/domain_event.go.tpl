{{- define "domainEvents" -}}
{{- if .Schema.Events }}
	Events: []*skel.EventSchema{
		{{- range $event := .Schema.Events }}
		{
			Name: {{ quote $event.Name }},
			SkelName: {{ quote $event.SkelName }},
			{{- if $event.Description }}
			Description: {{ quote $event.Description }},
			{{- end }}
			Hash: {{ quote $event.Hash }},
			Pub: {{ $event.Pub }},
			{{- if $event.Members }}
			Members: []*skel.MemberSchema{
				{{- range $member := $event.Members }}
				{
					Name: {{ quote $member.Name }},
					{{- if $member.Description }}
					Description: {{ quote $member.Description }},
					{{- end }}
					{{- if $member.Example }}
					Example: {{ quote $member.Example }},
					{{- end }}
					Type: {{ template "typeSchema" $member.Type }},
				},
				{{- end }}
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
