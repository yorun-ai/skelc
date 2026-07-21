package {{ $.PackageName }}

{{ if $.UsesPermissionCode -}}
import (
	"go.yorun.ai/vine/core/skel"
	"{{ $.PubImport.Path }}"
)
{{ else -}}
import "{{ $.PubImport.Path }}"
{{ end }}

{{ if $.Enums -}}
{{ range $e := $.Enums -}}
type {{ $e.Name }} = {{ $.PubPackageName }}.{{ $e.Name }}

const (
	{{ $e.UnspecifiedItem.Name }} = {{ $.PubPackageName }}.{{ $e.UnspecifiedItem.Name }}
{{- range $ei := $e.Items }}
	{{ $ei.Name }} = {{ $.PubPackageName }}.{{ $ei.Name }}
{{- end }}
)
{{ end -}}
{{ end }}
{{ if $.Data -}}
{{ range $s := $.Data -}}
type {{ $s.FullName }} = {{ $.PubPackageName }}.{{ $s.ReceiverType }}
{{ end -}}
{{ end }}
{{ if $.Configs -}}
{{ range $s := $.Configs -}}
type {{ $s.FullName }} = {{ $.PubPackageName }}.{{ $s.ReceiverType }}
{{ end -}}
{{ end }}
{{ if or $.Actors $.AuthCredentialData $.AuthServices -}}
{{ range $actor := $.Actors -}}
type {{ $actor.Name }} = {{ $.PubPackageName }}.{{ $actor.Name }}
{{ end -}}
{{ range $s := $.AuthCredentialData -}}
type {{ $s.FullName }} = {{ $.PubPackageName }}.{{ $s.ReceiverType }}
{{ end -}}
{{ range $service := $.AuthServices -}}
type {{ $service.ServerName }} = {{ $.PubPackageName }}.{{ $service.ServerName }}
type {{ $service.ERServerName }} = {{ $.PubPackageName }}.{{ $service.ERServerName }}
type {{ $service.DefaultServerName }} = {{ $.PubPackageName }}.{{ $service.DefaultServerName }}
type {{ $service.DefaultERServerName }} = {{ $.PubPackageName }}.{{ $service.DefaultERServerName }}
{{ end -}}
{{ end }}
{{ if $.Resources -}}
{{ range $resource := $.Resources -}}
{{ if $resource.Actions -}}
const (
{{- range $action := $resource.Actions }}
	{{ $action.PermissionName }} = {{ $.PubPackageName }}.{{ $action.PermissionName }}
{{- end }}
)

func {{ $resource.PermissionCodesName }}() []skel.PermissionCode {
	return {{ $.PubPackageName }}.{{ $resource.PermissionCodesName }}()
}

{{ end -}}
{{ end -}}
{{ end }}
{{ if $.Services -}}
{{ range $service := $.Services -}}
type {{ $service.ClientName }} = {{ $.PubPackageName }}.{{ $service.ClientName }}
type {{ $service.ERClientName }} = {{ $.PubPackageName }}.{{ $service.ERClientName }}

var {{ $service.ClientCtorName }} = {{ $.PubPackageName }}.{{ $service.ClientCtorName }}
var {{ $service.ERClientCtorName }} = {{ $.PubPackageName }}.{{ $service.ERClientCtorName }}
{{ end -}}
{{ end }}
{{ if $.Events -}}
{{ range $event := $.Events -}}
type {{ $event.ListenerName }} = {{ $.PubPackageName }}.{{ $event.ListenerName }}
type {{ $event.DefaultListenerName }} = {{ $.PubPackageName }}.{{ $event.DefaultListenerName }}
{{ end -}}
{{ end -}}
