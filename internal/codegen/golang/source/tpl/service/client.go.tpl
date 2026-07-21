{{- define "serviceClient" -}}{{ if not .ServerOnly }}
// {{ .Name }} / Client

type {{ .ClientName }} interface { {{- range .Methods }}
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
	_ivOpts ...rpc.InvokeOption){{ if .ResultType }} {{ .ResultType.Plain }}{{ end }}{{ end }}
}

type {{ .ClientImplName }} struct {
	clientER {{ .ERClientName }}
}

func {{ .ClientCtorName }}(clientER {{ .ERClientName }}) {{ .ClientName }} {
	return &{{ .ClientImplName }}{clientER: clientER}
}
{{ range .Methods }}
func (client *{{ $.ClientImplName }}) {{ .Name }}({{ range .Arguments }}{{ .Name }} {{ .Type.Plain }}, {{ end -}}
_ivOpts ...rpc.InvokeOption){{ if .ResultType }} {{ .ResultType.Plain }}{{ end }} {
	{{ if .ResultType }}ret, {{ end }}err := client.clientER.{{ .Name }}({{ range .Arguments }}{{ .Name }}, {{ end }}_ivOpts...)
	ex.PanicIfError(err){{ if .ResultType }}
	return ret{{ end }}
}
{{ end }}
{{- end }}{{- end -}}
