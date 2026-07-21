package source

import (
	"embed"
	"go/format"
	"strings"

	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

//go:embed tpl
var templateFS embed.FS

func loadTemplate(name string) string {
	content, err := templateFS.ReadFile("tpl/" + name)
	checkutil.CheckNilError(err, "read go template %s failed", name)
	return string(content)
}

func loadGoTemplate(name string) string {
	return joinTemplates("imports.go.tpl", name)
}

func joinTemplates(names ...string) string {
	templates := make([]string, 0, len(names))
	for _, name := range names {
		templates = append(templates, loadTemplate(name))
	}
	return strings.Join(templates, "\n")
}

func (g *_Gen) renderGo(file string, tpl string, data any) {
	content := codegen.RenderTemplate(tpl, data)
	formatted, err := format.Source([]byte(content))
	checkutil.CheckNilError(err, "format generated %s failed", file)
	g.Renderer.Write(file, string(formatted))
}
