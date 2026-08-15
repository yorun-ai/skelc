{{- define "dataClone" -}}
{{- if .Clone }}

{{- if .CloneParameters }}
// {{ .CloneMethodName }} returns a copy whose value isolation depends on each clone callback.
// Passing an identity callback does not guarantee isolation for reference-backed values.
{{- else }}
// {{ .CloneMethodName }} returns a value-isolated copy of the generated data.
{{- end }}
func (v {{ .ReceiverType }}) {{ .CloneMethodName }}({{ template "goParameters" .CloneParameters }}) {{ .ReceiverType }} {
	cloned := v
{{ template "goBlock" .CloneBlock -}}
	return cloned
}
{{- end }}
{{- end -}}
