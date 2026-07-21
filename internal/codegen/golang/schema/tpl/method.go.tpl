{{- define "methodSchema" -}}
&skel.MethodSchema{
	Name: {{ quote .Name }},
	SkelName: {{ quote .SkelName }},
	{{- if .Description }}
	Description: {{ quote .Description }},
	{{- end }}
	Hash: {{ quote .Hash }},
	{{- if .Example }}
	Example: {{ quote .Example }},
	{{- end }}
	AuthMode: {{ authLiteral .AuthMode }},
	{{- if .Require }}
	Require: {{ template "permissionRequire" .Require }},
	{{- end }}
	{{- if .InputDescription }}
	InputDescription: {{ quote .InputDescription }},
	{{- end }}
	{{- if .OutputDescription }}
	OutputDescription: {{ quote .OutputDescription }},
	{{- end }}
	{{- if .OutputExample }}
	OutputExample: {{ quote .OutputExample }},
	{{- end }}
	{{- if .Arguments }}
	Arguments: []*skel.MemberSchema{
		{{- range $argument := .Arguments }}
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
	{{- if .ResultType }}
	ResultType: {{ template "typeSchema" .ResultType }},
	{{- end }}
}
{{- end }}
