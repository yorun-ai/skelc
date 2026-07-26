{{- define "serviceSchema" -}}
&skel.ServiceSchema{
	Name: {{ quote .Name }},
	SkelName: {{ quote .SkelName }},
	{{- if .Description }}
	Description: {{ quote .Description }},
	{{- end }}
	Hash: {{ quote .Hash }},
	Pub: {{ .Pub }},
	AuthMode: {{ authLiteral .AuthMode }},
	{{- if .Audiences }}
	Audiences: []*skel.ActorAudienceSchema{
		{{- range $actor := .Audiences }}
		{Name: {{ quote $actor.Name }}, SkelName: {{ quote $actor.SkelName }}{{ with $actor.Via }}, Via: {{ viaLiteral . }}{{ end }}},
		{{- end }}
	},
	{{- end }}
	{{- if .Require }}
	Require: {{ template "permissionRequire" .Require }},
	{{- end }}
	{{- if .Methods }}
	Methods: []*skel.MethodSchema{
		{{- range $method := .Methods }}
		{{ template "methodSchemaValue" $method }},
		{{- end }}
	},
	{{- end }}
}
{{- end }}
