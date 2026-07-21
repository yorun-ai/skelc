{{- define "dataSchema" -}}
&skel.DataSchema{
	Name: {{ quote .Name }},
	SkelName: {{ quote .SkelName }},
	{{- if .Description }}
	Description: {{ quote .Description }},
	{{- end }}
	Hash: {{ quote .Hash }},
	{{- if .TypeParameters }}
	TypeParameters: []string{
		{{- range $typeParameter := .TypeParameters }}
		{{ quote $typeParameter }},
		{{- end }}
	},
	{{- end }}
	{{- if .Members }}
	Members: []*skel.MemberSchema{
		{{- range $member := .Members }}
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
}
{{- end }}
