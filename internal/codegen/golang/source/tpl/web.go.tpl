package {{ $.PackageName }}{{ template "imports" . }}

func init() {
{{- range $web := $.Webs }}
	web.Register({{ $web.SpecName }})
{{- end }}
}
{{ range $web := $.Webs }}
{{- if $web.CommentLines }}
{{- range $line := $web.CommentLines }}
// {{ $line }}
{{- end }}
{{- end }}

var {{ $web.SpecName }} = &web.WebSpec{
	Name: "{{ $web.Name }}",
	SkelName: "{{ $web.SkelName }}",
	Hash: "{{ $web.Hash }}",
	ServerType: reflect.TypeFor[{{ $web.ServerName }}](),
	DefaultServerType: reflect.TypeFor[*{{ $web.DefaultServerName }}](),
}

type {{ $web.ServerName }} interface {
	web.Handler

	mustBe{{ $web.ServerName }}()
}

type {{ $web.DefaultServerName }} struct {
}

func (*{{ $web.DefaultServerName }}) Routes(*web.Router) {
	panic("method routes is not implemented")
}

func (*{{ $web.DefaultServerName }}) mustBe{{ $web.ServerName }}() {}
{{ end }}
