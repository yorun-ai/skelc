domain {{ .Domain.Name }}
{{ if .Imports }}
{{ range $import := .Imports -}}
import {{ $import.Name }}{{ with importAlias $import }} as {{ . }}{{ end }}
{{ end -}}
{{ end }}
{{ range $i, $event := .Events -}}
{{ if $i }}
{{ end -}}
{{ template "description" (description $event.Description 0) }}pub event {{ $event.Name }} {
    payload {
{{- range $member := $event.Members }}
{{ template "description" (description $member.Description 8) }}{{ template "example" (example $member.Example 8) }}        {{ $member.Name }}: {{ template "type" (typeRef $member.Type) }}
{{- end }}
    }
}
{{ end -}}
