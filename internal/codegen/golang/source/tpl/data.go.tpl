package {{ $.PackageName }}{{ template "imports" . }}
{{ range $s := $.Data }}
{{- if $s.CommentLines }}
{{- range $line := $s.CommentLines }}
// {{ $line }}
{{- end }}
{{- end }}
{{ if not $s.Members -}}
type {{ $s.FullName }} struct{}
{{- else -}}
type {{ $s.FullName }} struct { {{ range $sm := $s.Members }}
	{{- if $sm.CommentLines }}
	{{- range $line := $sm.CommentLines }}
	// {{ $line }}
	{{- end }}
	{{- end }}
	{{ $sm.Name }} {{ $sm.Type.Plain }} `json:"{{ $sm.SkelName }}"`{{ end }}
}
{{- end }}
{{- if $s.Validate }}

func (v *{{ $s.ReceiverType }}) Validate(path string) error {
	if err := rpc.CheckValueNotNil(v, path); err != nil {
		return err
	}
{{- range $line := $s.CheckLines }}
{{ $line }}
{{- end }}
	return nil
}
{{- end }}
{{ end }}
