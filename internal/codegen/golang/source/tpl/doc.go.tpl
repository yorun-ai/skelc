{{- if .CommentLines }}{{- range $line := .CommentLines }}{{ printf "// %s\n" $line }}{{- end }}{{- else }}{{ printf "// %s\n" .PackageName }}{{- end -}}
package {{ .PackageName }}
