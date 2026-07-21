domain {{ .Domain.Name }}
{{ if .Imports }}
{{ range $import := .Imports -}}
import {{ $import.Name }}{{ with importAlias $import }} as {{ . }}{{ end }}
{{ end -}}
{{ end }}
{{ range $i, $actor := .Actors -}}
{{ if $i }}
{{ end -}}
{{ template "description" (description $actor.Description 0) }}pub actor {{ $actor.Name }} {
{{- range $via := $actor.Vias }}
    via {{ $via.Name }} {}
{{- end }}
{{- if $actor.AuthEnabled }}
    auth {
{{- with $actor.AuthCredential }}
        credential {
{{- range $member := .Members }}
{{ template "description" (description $member.Description 12) }}{{ template "example" (example $member.Example 12) }}            {{ $member.Name }}: {{ template "type" (typeRef $member.Type) }}
{{- end }}
        }
{{- end }}
{{- with $actor.AuthInfo }}
        info {
{{- range $member := .Members }}
{{ template "description" (description $member.Description 12) }}{{ template "example" (example $member.Example 12) }}            {{ $member.Name }}: {{ template "type" (typeRef $member.Type) }}
{{- end }}
        }
{{- end }}
    }
{{- end }}
{{- if $actor.PermEnabled }}
    permission {}
{{- end }}
}
{{ end -}}
