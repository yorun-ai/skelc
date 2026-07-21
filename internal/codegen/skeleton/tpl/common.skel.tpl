{{ define "description" }}{{ with . }}{{ $indent := spaces " " .Indent }}{{ if .Multiline }}{{ $indent }}@desc("""
{{ range .Lines }}{{ $indent }}{{ . }}
{{ end }}{{ $indent }}""")
{{ else }}{{ $indent }}@desc({{ .Quoted }})
{{ end }}{{ end }}{{ end }}
{{ define "example" }}{{ with . }}{{ $indent := spaces " " .Indent }}{{ if .Multiline }}{{ if .Object }}{{ $indent }}@example({
{{ range .Lines }}{{ $indent }}    {{ . }}
{{ end }}{{ $indent }}})
{{ else }}{{ $indent }}@example(
{{ range .Lines }}{{ $indent }}{{ . }}
{{ end }}{{ $indent }})
{{ end }}{{ else }}{{ $indent }}@example({{ .Quoted }})
{{ end }}{{ end }}{{ end }}
{{ define "type" }}{{ if eq .Kind "list" }}list<{{ template "type" .Value }}>{{ else if eq .Kind "map" }}map<{{ template "type" .Key }}, {{ template "type" .Value }}>{{ else }}{{ with .Qualifier }}{{ . }}.{{ end }}{{ .Name }}{{ with .Arguments }}<{{ range $i, $argument := . }}{{ if $i }}, {{ end }}{{ template "type" $argument }}{{ end }}>{{ end }}{{ end }}{{ if .Nullable }}?{{ end }}{{ end }}
