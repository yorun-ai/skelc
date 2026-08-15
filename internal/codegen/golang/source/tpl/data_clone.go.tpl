{{- define "dataClone" -}}
{{- if .Clone }}

// {{ .CloneMethodName }} returns a value-isolated copy of the generated data.
{{- if .CloneParameters }}
// Each clone callback must return a value-isolated copy of its argument.
{{- end }}
func (v {{ .ReceiverType }}) {{ .CloneMethodName }}({{ template "goParameters" .CloneParameters }}) {{ .ReceiverType }} {
	cloned := v
{{ template "goBlock" .CloneBlock -}}
	return cloned
}
{{- end }}
{{- end -}}
