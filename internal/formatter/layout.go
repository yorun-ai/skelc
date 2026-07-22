package formatter

import (
	"bytes"
	"strings"
)

const indentWidth = 4

type _LayoutKind uint8

const (
	layoutText _LayoutKind = iota
	layoutRaw
	layoutSpace
	layoutLine
	layoutBlankLine
	layoutIndent
	layoutDedent
)

type _Layout struct {
	kind       _LayoutKind
	text       string
	baseIndent int
}

type _Renderer struct {
	output       bytes.Buffer
	indent       int
	lineStart    bool
	pendingBreak int
}

func render(layout []_Layout) []byte {
	renderer := _Renderer{lineStart: true}
	for _, item := range layout {
		switch item.kind {
		case layoutText:
			renderer.writeText(item.text)
		case layoutRaw:
			renderer.writeRaw(item.text, item.baseIndent)
		case layoutSpace:
			renderer.writeSpace()
		case layoutLine:
			renderer.requestBreak(1)
		case layoutBlankLine:
			renderer.requestBreak(2)
		case layoutIndent:
			renderer.indent++
		case layoutDedent:
			renderer.indent = max(0, renderer.indent-1)
		}
	}
	return finishSource(renderer.output.Bytes())
}

func (r *_Renderer) writeText(value string) {
	r.flushBreak()
	r.writeIndent()
	r.output.WriteString(value)
}

func (r *_Renderer) writeRaw(value string, baseIndent int) {
	r.flushBreak()
	r.writeIndent()
	if baseIndent < 0 {
		r.output.WriteString(value)
		r.lineStart = strings.HasSuffix(value, "\n")
		return
	}
	lines := strings.Split(value, "\n")
	r.output.WriteString(lines[0])
	indent := strings.Repeat(" ", r.indent*indentWidth)
	for _, line := range lines[1:] {
		r.output.WriteByte('\n')
		line = trimBaseIndent(line, baseIndent)
		if line != "" {
			r.output.WriteString(indent)
			r.output.WriteString(line)
		}
	}
	r.lineStart = len(lines) > 1 && lines[len(lines)-1] == ""
}

func trimBaseIndent(line string, width int) string {
	index := 0
	for index < len(line) && index < width && (line[index] == ' ' || line[index] == '\t') {
		index++
	}
	return line[index:]
}

func (r *_Renderer) writeSpace() {
	if r.pendingBreak > 0 || r.lineStart || r.output.Len() == 0 {
		return
	}
	data := r.output.Bytes()
	if data[len(data)-1] != ' ' && data[len(data)-1] != '\n' {
		r.output.WriteByte(' ')
	}
}

func (r *_Renderer) requestBreak(lines int) {
	if r.output.Len() == 0 {
		return
	}
	r.pendingBreak = max(r.pendingBreak, lines)
}

func (r *_Renderer) flushBreak() {
	if r.pendingBreak == 0 {
		return
	}
	data := r.output.Bytes()
	for len(data) > 0 && (data[len(data)-1] == ' ' || data[len(data)-1] == '\t') {
		r.output.Truncate(len(data) - 1)
		data = r.output.Bytes()
	}
	if len(data) > 0 && data[len(data)-1] == '\n' {
		r.pendingBreak--
	}
	for range r.pendingBreak {
		r.output.WriteByte('\n')
	}
	r.pendingBreak = 0
	r.lineStart = true
}

func (r *_Renderer) writeIndent() {
	if !r.lineStart {
		return
	}
	r.output.WriteString(strings.Repeat(" ", r.indent*indentWidth))
	r.lineStart = false
}
