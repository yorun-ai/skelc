{{- define "domainConfigs" -}}
{{- if .Schema.Configs }}
	Configs: []*skel.ConfigSchema{
		{{- range $config := .Schema.Configs }}
		{
			Name: {{ quote $config.Name }},
			SkelName: {{ quote $config.SkelName }},
			{{- if $config.Description }}
			Description: {{ quote $config.Description }},
			{{- end }}
			Hash: {{ quote $config.Hash }},
			Pub: {{ $config.Pub }},
			{{- if $config.Lifecycle }}
			Lifecycle: {{ quote $config.Lifecycle }},
			{{- end }}
			{{- if $config.Members }}
			Members: []*skel.MemberSchema{
				{{- range $member := $config.Members }}
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
