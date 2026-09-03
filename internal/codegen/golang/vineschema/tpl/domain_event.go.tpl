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
			{{- template "deprecatedFields" $event }}
			Hash: {{ quote $event.Hash }},
			Pub: {{ $event.Pub }},
			{{- if $event.Sensitive }}
			Sensitive: true,
			{{- end }}
			{{- if $event.Members }}
			Members: []*skel.MemberSchema{
				{{- range $member := $event.Members }}
				{{ template "memberSchema" $member }},
				{{- end }}
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
{{- end }}
