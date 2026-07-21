package skeleton

import (
	"bytes"
	"embed"
	"strings"
	"text/template"

	"go.yorun.ai/skelc/internal/formatter"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

//go:embed tpl/*.skel.tpl
var skelTemplateFS embed.FS

func newSkelTemplate() *template.Template {
	tpl, err := template.New("skel").Funcs(skelTemplateFuncs()).ParseFS(skelTemplateFS, "tpl/*.skel.tpl")
	checkutil.CheckNilError(err, "parse skel templates failed")
	return tpl
}

func (g *_Gen) renderSkel(templateName string, payload *_SkelPayload) string {
	var rendered bytes.Buffer
	err := g.template.ExecuteTemplate(&rendered, templateName, payload)
	checkutil.CheckNilError(err, "execute skel template %s failed", templateName)
	return string(formatter.Source(rendered.Bytes()))
}

func skelTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"description":  descriptionView,
		"example":      exampleView,
		"importAlias":  importAlias,
		"emptyMethod":  emptyMethod,
		"authMarker":   authMarker,
		"methodAuth":   methodAuthMarker,
		"typeParams":   typeParameterNames,
		"typeRef":      typeView,
		"checkArgs":    renderResourceCheckArguments,
		"configSuffix": configLifecycle,
		"spaces":       strings.Repeat,
	}
}
