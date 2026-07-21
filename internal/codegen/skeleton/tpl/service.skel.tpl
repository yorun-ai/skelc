domain {{ .Domain.Name }}
{{ if .Imports }}
{{ range $import := .Imports -}}
import {{ $import.Name }}{{ with importAlias $import }} as {{ . }}{{ end }}
{{ end -}}
{{ end }}
{{ range $i, $service := .Services -}}
{{ if $i }}
{{ end -}}
{{- $serviceAuth := authMarker $service.Auth -}}
{{ template "description" (description $service.Description 0) }}pub service {{ $service.Name }} {
{{- range $audience := $service.Audiences }}
    for {{ $audience.Actor }}{{ with $audience.Via }} via {{ . }}{{ end }}
{{- end }}
{{- if and $service.Audiences (or $serviceAuth $service.Methods) }}

{{ end -}}
{{- if $serviceAuth }}
    {{ $serviceAuth }}
{{- end }}
{{- if and $serviceAuth $service.Methods }}

{{ end }}
{{- range $i, $method := $service.Methods }}
{{- if $i }}

{{ end }}
{{- $methodAuth := methodAuth $method -}}
{{ template "description" (description $method.Description 4) }}{{ if emptyMethod $method $service }}    method {{ $method.Name }} {}
{{- else }}    method {{ $method.Name }} {
{{- if $methodAuth }}
        {{ $methodAuth }}
{{- end }}
{{- if and $methodAuth (or $method.Arguments $method.ResultType) }}

{{ end }}
{{- if $method.Arguments }}
{{ template "description" (description $method.InputDescription 8) }}        input {
{{- range $argument := $method.Arguments }}
{{ template "description" (description $argument.Description 12) }}{{ template "example" (example $argument.Example 12) }}            {{ $argument.Name }}: {{ template "type" (typeRef $argument.Type) }}
{{- end }}
        }
{{- end }}
{{- if $method.ResultType }}
{{ template "description" (description $method.OutputDescription 8) }}{{ template "example" (example $method.OutputExample 8) }}        output {{ template "type" (typeRef $method.ResultType) }}
{{- end }}
    }
{{- end }}
{{- end }}
}
{{ end -}}
