{{- define "domainEnums" -}}
{{- if .Schema.Enums }}
	Enums: []*skel.EnumSchema{
		{{- range $enum := .Schema.Enums }}
		{
			Name: {{ quote $enum.Name }},
			SkelName: {{ quote $enum.SkelName }},
			{{- if $enum.Description }}
			Description: {{ quote $enum.Description }},
			{{- end }}
			Hash: {{ quote $enum.Hash }},
			{{- if $enum.Items }}
			Items: []*skel.EnumItemSchema{
				{{- range $item := $enum.Items }}
				{
					Name: {{ quote $item.Name }},
					{{- if $item.Description }}
					Description: {{ quote $item.Description }},
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
