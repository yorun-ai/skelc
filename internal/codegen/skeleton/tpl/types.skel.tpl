domain {{ .Domain.Name }}
{{ if .Imports }}
{{ range $import := .Imports -}}
import {{ $import.Name }}{{ with importAlias $import }} as {{ . }}{{ end }}
{{ end -}}
{{ end }}
{{ range $i, $enum := .Enums -}}
{{ if $i }}
{{ end -}}
{{ template "description" (description $enum.Description 0) }}pub enum {{ $enum.Name }} {
{{- range $item := $enum.Items }}
{{ template "description" (description $item.Description 4) }}    {{ $item.Name }}
{{- end }}
}
{{ end -}}
{{ range $i, $data := .Data -}}
{{ if or $.Enums $i }}
{{ end -}}
{{ template "description" (description $data.Description 0) }}pub data {{ $data.Name }}{{ with typeParams $data.TypeParameters }}<{{ range $i, $name := . }}{{ if $i }}, {{ end }}{{ $name }}{{ end }}>{{ end }} {
{{- range $member := $data.Members }}
{{ template "description" (description $member.Description 4) }}{{ template "example" (example $member.Example 4) }}    {{ $member.Name }}: {{ template "type" (typeRef $member.Type) }}
{{- end }}
}
{{ end -}}
{{ range $i, $config := .Configs -}}
{{ if or $.Enums $.Data $i }}
{{ end -}}
{{ template "description" (description $config.Description 0) }}pub config {{ $config.Name }}{{ with configSuffix $config }} {{ . }}{{ end }} {
{{- range $member := $config.Members }}
{{ template "description" (description $member.Description 4) }}{{ template "example" (example $member.Example 4) }}    {{ $member.Name }}: {{ template "type" (typeRef $member.Type) }}
{{- end }}
}
{{ end -}}
{{ range $i, $resource := .Resources -}}
{{ if or $.Enums $.Data $.Configs $i }}
{{ end -}}
{{ template "description" (description $resource.Description 0) }}{{ if $resource.Pub }}pub {{ end }}resource {{ $resource.Name }} {
{{ range $check := $resource.Checks }}    check {{ $check.Name }}({{ range $i, $arg := checkArgs $check }}{{ if $i }}, {{ end }}{{ $arg.Name }}: {{ template "type" (typeRef $arg.Type) }}{{ end }})
{{ end }}
{{ if and $resource.Checks $resource.Actions }}
{{ end -}}
{{ range $i, $action := $resource.Actions }}{{ if $i }}
{{ end }}
{{ template "description" (description $action.Description 4) }}    action {{ $action.Name }}{{ if $action.Checks }} {
{{ range $check := $action.Checks }}        check {{ $check.Name }}({{ range $i, $arg := checkArgs $check }}{{ if $i }}, {{ end }}{{ $arg.Name }}: {{ template "type" (typeRef $arg.Type) }}{{ end }})
{{- end }}
    }{{ end }}
{{- end }}
}
{{ end -}}
