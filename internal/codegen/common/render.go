package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"go.yorun.ai/skelc/internal/util/checkutil"
)

type Renderer struct {
	outputDir string
	err       error
}

func NewRenderer(outputDir string) *Renderer {
	return new(Renderer{outputDir: outputDir})
}

func (r *Renderer) Render(file string, tpl string, data any) {
	content := renderTemplate(tpl, data)
	r.Write(file, content)
}

func (r *Renderer) Write(file string, content string) {
	if r.err != nil {
		return
	}
	content = normalizeTrailingNewline(content)
	fullPath := filepath.Join(r.outputDir, file)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		r.err = fmt.Errorf("create output directory for %s: %w", fullPath, err)
		return
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		r.err = fmt.Errorf("write %s: %w", fullPath, err)
	}
}

func (r *Renderer) Err() error { return r.err }

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
