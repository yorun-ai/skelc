package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"go.yorun.ai/skelc/internal/util/checkutil"
)

type Renderer struct {
	outputDir string
}

func NewRenderer(outputDir string) *Renderer {
	return new(Renderer{outputDir: outputDir})
}

func (r *Renderer) Render(file string, tpl string, data any) {
	content := renderTemplate(tpl, data)
	r.Write(file, content)
}

func (r *Renderer) Write(file string, content string) {
	content = normalizeTrailingNewline(content)
	fullPath := filepath.Join(r.outputDir, file)
	err := os.MkdirAll(filepath.Dir(fullPath), 0o755)
	checkutil.CheckNilError(err, "create output directory for %s failed", fullPath)
	err = os.WriteFile(fullPath, []byte(content), 0o644)
	checkutil.CheckNilError(err, "write %s failed", fullPath)
}

func RenderTemplate(tplString string, payloadData any) string {
	return RenderTemplateWithFuncs(tplString, payloadData, nil)
}

func RenderTemplateWithFuncs(tplString string, payloadData any, funcs template.FuncMap) string {
	tpl := template.New("template")
	if funcs != nil {
		tpl = tpl.Funcs(funcs)
	}
	tpl, err := tpl.Parse(tplString)
	checkutil.CheckNilError(err, "parse template failed")

	var rendered strings.Builder
	err = tpl.Execute(&rendered, payloadData)
	checkutil.CheckNilError(err, "execute template failed")

	return rendered.String()
}

func renderTemplate(tplString string, payloadData any) string {
	return RenderTemplateWithFuncs(tplString, payloadData, nil)
}

func normalizeTrailingNewline(content string) string {
	content = strings.TrimLeft(content, "\n")
	return strings.TrimRight(content, "\n") + "\n"
}
