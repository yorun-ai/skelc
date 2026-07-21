{{- define "serviceArguments" -}}{{ if .HasMethodArguments }}
// {{ .Name }} / Arguments
{{ range .Methods }}{{ if .ArgumentsData }}
type {{ .ArgumentsData.Name }} struct {
{{- range $index, $argument := .ArgumentsData.Members }}
	{{ $argument.Name }} {{ $argument.Type.Plain }} `json:"{{ $argument.SkelName }}" arg:"{{ $index }}"`
{{- end }}
}
{{ end }}{{- end -}}
{{ end }}{{ end }}
