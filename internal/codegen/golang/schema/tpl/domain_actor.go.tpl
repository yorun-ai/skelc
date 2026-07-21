{{- define "domainActors" -}}
{{- if .Schema.Actors }}
	Actors: []*skel.ActorSchema{
		{{- range $actor := .Schema.Actors }}
		{
			Name: {{ quote $actor.Name }},
			SkelName: {{ quote $actor.SkelName }},
			{{- if $actor.Description }}
			Description: {{ quote $actor.Description }},
			{{- end }}
			Hash: {{ quote $actor.Hash }},
			{{- if $actor.Vias }}
			Vias: []skel.ActorVia{
				{{- range $via := $actor.Vias }}
				{{ viaLiteral $via }},
				{{- end }}
			},
			{{- end }}
			AuthEnabled: {{ $actor.AuthEnabled }},
			{{- if $actor.AuthCredential }}
			AuthCredential: {{ template "dataSchema" $actor.AuthCredential }},
			{{- end }}
			{{- if $actor.AuthInfo }}
			AuthInfo: {{ template "dataSchema" $actor.AuthInfo }},
			{{- end }}
			{{- if $actor.AuthService }}
			AuthService: {{ template "serviceSchema" $actor.AuthService }},
			{{- end }}
			{{- if $actor.AuthMethod }}
			AuthMethod: {{ template "methodSchema" $actor.AuthMethod }},
			{{- end }}
			PermEnabled: {{ $actor.PermEnabled }},
			{{- if $actor.PermService }}
			PermService: {{ template "serviceSchema" $actor.PermService }},
			{{- end }}
			{{- if $actor.PermMethod }}
			PermMethod: {{ template "methodSchema" $actor.PermMethod }},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
