{{- define "serviceERClient" -}}{{ if not .ServerOnly }}
// {{ .Name }} / ERClient

type {{ .ERClientName }} interface { {{- range .Methods }}
	{{- if .CommentLines }}
	{{- range $lineIndex, $line := .CommentLines }}
	{{- if eq $lineIndex 0 }}
	// {{ $line }}
	{{- else }}
	//   {{ $line }}
	{{- end }}
	{{- end }}
	{{- end }}
	{{ .Name }}(
	{{- range .Arguments }}{{ .Name }} {{ .Type.Plain }}, {{ end -}}
	_ivOpts ...rpc.InvokeOption){{ if .ResultType }} ({{ .ResultType.Plain }}, ex.Error){{ else }} ex.Error{{ end }}{{ end }}
}

type {{ .ERClientImplName }} struct {
	rpcClient *rpc.Client
}

func {{ .ERClientCtorName }}(rpcClient *rpc.Client) {{ .ERClientName }} {
	return &{{ .ERClientImplName }}{
		rpcClient: rpcClient,
	}
}
{{ range .Methods }}
func (client *{{ $.ERClientImplName }}) {{ .Name }}({{ range .Arguments }}{{ .Name }} {{ .Type.Plain }}, {{ end -}}
_ivOpts ...rpc.InvokeOption) {{ if .ResultType }}({{ .ResultType.Plain }}, ex.Error){{ else }}ex.Error{{ end }} {
	{{ if .ResultType }}retI{{ else }}_{{ end }}, errI := client.rpcClient.Invoke({{ .SpecName }}.Info(), {{ if .ArgumentsData }}&{{ .ArgumentsData.Name }}{ {{ range .Arguments }}
		{{ .MemberName }}: {{ .Name }},{{ end }}
	{{ "}" }}{{ else }}nil{{ end }}, _ivOpts...){{ if .ResultType }}
	ret, _ := retI.({{ .ResultType.Plain }}){{ end }}
	err, _ := errI.(ex.Error)
	return {{ if .ResultType }}ret, {{ end }}err
}
{{ end }}
{{- end }}{{- end -}}
