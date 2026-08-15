{{- define "goParameters" -}}
{{- range $index, $parameter := . -}}
{{- if $index }}, {{ end -}}
{{ $parameter.Name }} {{ $parameter.Type }}
{{- end -}}
{{- end -}}

{{- define "goExpressionList" -}}
{{- range $index, $expression := . -}}
{{- if $index }}, {{ end -}}
{{ template "goExpression" $expression }}
{{- end -}}
{{- end -}}

{{- define "goExpression" -}}
{{- if .Raw -}}
{{ .Raw }}
{{- else if .Call -}}
{{ template "goExpression" .Call.Function }}({{ template "goExpressionList" .Call.Arguments }})
{{- else if .Function -}}
{{ template "goFunction" .Function }}
{{- end -}}
{{- end -}}

{{- define "goAssignment" -}}
{{- range $index, $target := .Targets -}}
{{- if $index }}, {{ end -}}
{{ $target }}
{{- end }} {{ .Operator }} {{ template "goExpressionList" .Values }}
{{- end -}}

{{- define "goBlock" -}}
{{- range .Statements -}}
{{ template "goStatement" . }}
{{ end -}}
{{- end -}}

{{- define "goStatement" -}}
{{- if .Assignment -}}
{{ template "goAssignment" .Assignment }}
{{- else if .Expression -}}
{{ template "goExpression" .Expression }}
{{- else if .If -}}
if {{ if .If.Init }}{{ template "goAssignment" .If.Init }}; {{ end }}{{ template "goExpression" .If.Condition }} {
{{ template "goBlock" .If.Then -}}
}{{ if .If.Else }} else {
{{ template "goBlock" .If.Else -}}
}{{ end }}
{{- else if .Range -}}
for {{ range $index, $name := .Range.Names }}{{ if $index }}, {{ end }}{{ $name }}{{ end }} := range {{ template "goExpression" .Range.Source }} {
{{ template "goBlock" .Range.Body -}}
}
{{- else if .Return -}}
return{{ if .Return.Values }} {{ template "goExpressionList" .Return.Values }}{{ end }}
{{- else if .Variable -}}
var {{ .Variable.Name }} {{ .Variable.Type }}
{{- end -}}
{{- end -}}

{{- define "goFunction" -}}
func({{ template "goParameters" .Parameters }}){{ if .Result }} {{ .Result }}{{ end }} {
{{ template "goBlock" .Body -}}
}
{{- end -}}
