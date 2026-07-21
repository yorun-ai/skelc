{{- range $import := $.TypeImports }}
import type * as {{ $import.Alias }} from '{{ $import.Path }}';
{{- end }}
{{- if $.TypeImports }}

{{- end }}
{{- range $index, $e := $.Enums }}
{{- if gt $index 0 }}

{{- end }}
{{- if $e.CommentLines }}
/**
{{- range $line := $e.CommentLines }}
 * {{ $line }}
{{- end }}
 */
{{- end }}
export type {{ $e.Name }} =
{{- range $item := $e.Items }}
  | {{ $item.Literal }}{{ $item.ValuePadding }}{{- if $item.CommentLines }} // {{ index $item.CommentLines 0 }}{{- end }}
{{- end }}
;
{{- end }}
{{- if and $.Enums $.Data }}

{{- end }}
{{- range $index, $s := $.Data }}
{{- if gt $index 0 }}

{{- end }}
{{- if $s.CommentLines }}
/**
{{- range $line := $s.CommentLines }}
 * {{ $line }}
{{- end }}
 */
{{- end }}
{{ if not $s.Members -}}
export type {{ $s.FullName }} = {}
{{- else -}}
export type {{ $s.FullName }} = {{ "{" }}{{ range $sm := $s.Members }}
  {{- if $sm.CommentLines }}
  /**
  {{- range $line := $sm.CommentLines }}
   * {{ $line }}
  {{- end }}
   */
  {{- end }}
  {{ $sm.Name }}:{{ $sm.NamePadding }} {{ $sm.Type.Plain }};{{ end }}
}
{{- end }}
{{- end }}
{{- if or $.Enums $.Data }}

{{- end }}
export {};
