{{- define "domainData" -}}
{{- if .Schema.Data }}
	Data: []*skel.DataSchema{
		{{- range $data := .Schema.Data }}
		{
			Name: {{ quote $data.Name }},
			SkelName: {{ quote $data.SkelName }},
			{{- if $data.Description }}
			Description: {{ quote $data.Description }},
			{{- end }}
			Hash: {{ quote $data.Hash }},
			{{- if $data.TypeParameters }}
			TypeParameters: []string{
				{{- range $typeParameter := $data.TypeParameters }}
				{{ quote $typeParameter }},
				{{- end }}
			},
			{{- end }}
			{{- if $data.Members }}
			Members: []*skel.MemberSchema{
				{{- range $member := $data.Members }}
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
