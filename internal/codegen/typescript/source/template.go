package source

import (
	"strings"

	"go.yorun.ai/skelc/internal/codegen"
)

func (g *_Gen) renderTs(file string, tpl string, data any) {
	content := codegen.RenderTemplate(tpl, data)
	g.renderer.Write(file, trimTrailingWhitespace(content))
}

func trimTrailingWhitespace(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
