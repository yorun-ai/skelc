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
			{{- template "deprecatedFields" $config }}
			Hash: {{ quote $config.Hash }},
			Pub: {{ $config.Pub }},
			{{- if $config.Sensitive }}
			Sensitive: true,
			{{- end }}
			{{- if $config.Lifecycle }}
			Lifecycle: {{ quote $config.Lifecycle }},
			{{- end }}
			{{- if $config.Members }}
			Members: []*skel.MemberSchema{
				{{- range $member := $config.Members }}
				{{ template "memberSchema" $member }},
				{{- end }}
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
