{{- define "permissionRequire" -}}
&skel.PermRequire{
	Expr: {{ template "permissionExpr" .Expr }},
}
{{- end }}

{{- define "permissionExpr" -}}
&skel.PermExpr{
	{{- template "permissionExprFields" . }}
}
{{- end }}

{{- define "permissionExprValue" -}}
{
	{{- template "permissionExprFields" . }}
}
{{- end }}

{{- define "permissionExprFields" }}
	Mode: {{ permissionRequireLiteral .Mode }},
	{{- if .Code }}
	Code: {{ quote .Code }},
	{{- end }}
	{{- if .Check }}
	Check: &skel.PermCheckInvocation{
		ResourceSkelName: {{ quote .Check.ResourceSkelName }},
		ActionName: {{ quote .Check.ActionName }},
		CheckName: {{ quote .Check.CheckName }},
		ServiceSkelName: {{ quote .Check.ServiceSkelName }},
		MethodSkelName: {{ quote .Check.MethodSkelName }},
		{{- if .Check.Arguments }}
		Arguments: []*skel.PermCheckArgument{
			{{- range $argument := .Check.Arguments }}
			{
				Name: {{ quote $argument.Name }},
				JsonPath: {{ quote $argument.JsonPath }},
				Type: {{ template "typeSchema" $argument.Type }},
			},
			{{- end }}
		},
		{{- end }}
	},
	{{- end }}
	{{- if .Children }}
	Children: []*skel.PermExpr{
		{{- range $child := .Children }}
		{{ template "permissionExprValue" $child }},
		{{- end }}
	},
	{{- end }}
{{- end }}
