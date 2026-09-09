package {{ .PackageName }}
{{ template "imports" . }}
func init() {
    {{ range $s := .Services }}vrpc.Register({{ .SpecName }})
    {{ range .Methods }}{
        method, ok := vrpc.GetMethodInfo("{{ $s.SkelName }}", "{{ .SkelName }}")
        if !ok {
            panic("missing generated vRPC method: {{ $s.SkelName }}/{{ .SkelName }}")
        }
        _{{ $s.Name }}{{ .Name }}Method = method
    }
    {{ end }}{{ end }}
}
{{ range .Services }}
{{ template "apiInfo" . }}
{{ template "serviceArguments" . }}
{{ template "apiClient" . }}
{{ end }}
