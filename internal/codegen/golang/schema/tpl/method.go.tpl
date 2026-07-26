{{- define "methodSchema" -}}
&skel.MethodSchema{{ template "methodSchemaValue" . }}
{{- end }}

{{- define "methodSchemaValue" -}}
{
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
	{{- if .ArgumentsSensitive }}
	ArgumentsSensitive: true,
	{{- end }}
	{{- if .OutputDescription }}
	OutputDescription: {{ quote .OutputDescription }},
	{{- end }}
	{{- if .OutputExample }}
	OutputExample: {{ quote .OutputExample }},
	{{- end }}
	{{- if .ResultSensitive }}
	ResultSensitive: true,
	{{- end }}
	{{- if .Arguments }}
	Arguments: []*skel.MemberSchema{
		{{- range $argument := .Arguments }}
		{{ template "memberSchema" $argument }},
		{{- end }}
	},
	{{- end }}
	{{- if .ResultType }}
	ResultType: {{ template "typeSchema" .ResultType }},
	{{- end }}
}
{{- end }}
