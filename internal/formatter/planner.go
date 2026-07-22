package formatter

import "strings"

type _TopLineKind uint8

const (
	topLineUnknown _TopLineKind = iota
	topLineDecorator
	topLineDomain
	topLineImport
	topLineDeclaration
	topLineComment
)

type _Planner struct {
	tokens         []_Token
	layout         []_Layout
	depth          int
	parenDepth     int
	lineTokenCount int
	previous       string
	topLine        _TopLineKind
	inRequire      bool
	parenIndents   []bool
	braceIndents   []bool
}

func buildLayout(tokens []_Token) []_Layout {
	planner := _Planner{tokens: tokens}
	for index, token := range tokens {
		planner.add(index, token)
	}
	return planner.layout
}

func (p *_Planner) add(index int, token _Token) {
	switch token.kind {
	case "Newline":
		p.addNewline(index, token.value)
	case "LineComment":
		p.addLineComment(token.value)
	case "BlockComment":
		p.addBlockComment(token)
	default:
		p.addSyntax(index, token)
	}
}

func (p *_Planner) addNewline(index int, value string) {
	breaks := 1
	if strings.Count(value, "\n") > 1 {
		breaks = 2
	}
	next := p.nextMeaningful(index + 1)
	if p.previous == "{" || next != nil && next.value == "}" {
		breaks = 1
	}
	if p.depth == 0 && p.parenDepth == 0 && p.canonicalTopLevelBreak(index) {
		breaks = 2
	}
	if p.inRequire && p.parenDepth == 0 {
		p.inRequire = false
	}
	p.breakLine(breaks)
}

func (p *_Planner) canonicalTopLevelBreak(index int) bool {
	next := p.nextMeaningful(index + 1)
	if next == nil {
		return false
	}
	nextKind := classifyTopLine(next.value, next.kind)
	if nextKind == topLineComment {
		return false
	}
	switch p.topLine {
	case topLineDecorator:
		return false
	case topLineImport:
		return nextKind != topLineImport
	case topLineDomain, topLineDeclaration:
		return true
	default:
		return false
	}
}

func (p *_Planner) addLineComment(value string) {
	if p.lineTokenCount > 0 {
		p.space()
	} else {
		p.startTopLine(value, "LineComment")
	}
	p.raw(value, 0)
	p.breakLine(1)
}

func (p *_Planner) addBlockComment(token _Token) {
	if p.lineTokenCount > 0 {
		p.space()
	} else {
		p.startTopLine(token.value, "BlockComment")
	}
	p.raw(token.value, max(0, token.pos.Column-1))
	p.previous = "comment"
	p.lineTokenCount++
}

func (p *_Planner) addSyntax(index int, token _Token) {
	value := token.value
	p.startTopLine(value, token.kind)

	switch value {
	case "{":
		p.before(value)
		p.text(value)
		p.previous = value
		p.lineTokenCount++
		if next := p.nextMeaningful(index + 1); next != nil && next.value == "}" {
			p.braceIndents = append(p.braceIndents, false)
			return
		}
		p.braceIndents = append(p.braceIndents, true)
		p.depth++
		p.layout = append(p.layout, _Layout{kind: layoutIndent})
		p.breakLine(1)
		return
	case "}":
		indented := false
		if len(p.braceIndents) > 0 {
			indented = p.braceIndents[len(p.braceIndents)-1]
			p.braceIndents = p.braceIndents[:len(p.braceIndents)-1]
		}
		if indented {
			p.depth--
			p.layout = append(p.layout, _Layout{kind: layoutDedent})
		}
		if p.previous != "{" {
			p.breakLine(1)
		}
		p.text(value)
		p.previous = value
		p.lineTokenCount++
		return
	case "(":
		p.before(value)
		p.text(value)
		p.parenDepth++
		indented := index+1 < len(p.tokens) && p.tokens[index+1].kind == "Newline"
		p.parenIndents = append(p.parenIndents, indented)
		if indented {
			p.layout = append(p.layout, _Layout{kind: layoutIndent})
		}
	case ")":
		if len(p.parenIndents) > 0 {
			indented := p.parenIndents[len(p.parenIndents)-1]
			p.parenIndents = p.parenIndents[:len(p.parenIndents)-1]
			if indented {
				p.layout = append(p.layout, _Layout{kind: layoutDedent})
			}
		}
		p.text(value)
		p.parenDepth = max(0, p.parenDepth-1)
	default:
		p.before(value)
		p.textOrRaw(token)
	}
	if value == "require" {
		p.inRequire = true
	}

	p.previous = value
	p.lineTokenCount++
}

func (p *_Planner) before(value string) {
	if p.lineTokenCount == 0 || noSpaceBefore(value) || noSpaceAfter(p.previous) {
		return
	}
	if p.previous == ":" && p.inRequire {
		return
	}
	p.space()
}

func (p *_Planner) startTopLine(value string, kind string) {
	if p.lineTokenCount > 0 || p.depth != 0 || p.parenDepth != 0 {
		return
	}
	p.topLine = classifyTopLine(value, kind)
}

func classifyTopLine(value string, kind string) _TopLineKind {
	switch {
	case kind == "LineComment" || kind == "BlockComment":
		return topLineComment
	case value == "@":
		return topLineDecorator
	case value == "domain":
		return topLineDomain
	case value == "import":
		return topLineImport
	case value == "pub" || isDeclarationKeyword(value):
		return topLineDeclaration
	default:
		return topLineUnknown
	}
}

func isDeclarationKeyword(value string) bool {
	switch value {
	case "enum", "data", "config", "actor", "resource", "service", "web", "event", "task":
		return true
	default:
		return false
	}
}

func noSpaceBefore(value string) bool {
	switch value {
	case "(", "[", "<", ")", "]", ">", ",", ".", "?", ":", ";":
		return true
	default:
		return false
	}
}

func noSpaceAfter(value string) bool {
	switch value {
	case "", "@", "(", "[", "<", ".", "*":
		return true
	default:
		return false
	}
}

func (p *_Planner) nextMeaningful(start int) *_Token {
	for index := start; index < len(p.tokens); index++ {
		if p.tokens[index].kind == "Newline" {
			continue
		}
		return &p.tokens[index]
	}
	return nil
}

func (p *_Planner) breakLine(lines int) {
	kind := layoutLine
	if lines > 1 {
		kind = layoutBlankLine
	}
	p.layout = append(p.layout, _Layout{kind: kind})
	p.previous = ""
	p.lineTokenCount = 0
}

func (p *_Planner) textOrRaw(token _Token) {
	if token.kind == "TripleString" {
		p.raw(token.value, max(0, token.pos.Column-1))
		return
	}
	if strings.Contains(token.value, "\n") {
		p.raw(token.value, -1)
		return
	}
	p.text(token.value)
}

func (p *_Planner) text(value string) {
	p.layout = append(p.layout, _Layout{kind: layoutText, text: value})
}

func (p *_Planner) raw(value string, baseIndent int) {
	p.layout = append(p.layout, _Layout{kind: layoutRaw, text: value, baseIndent: baseIndent})
}

func (p *_Planner) space() {
	p.layout = append(p.layout, _Layout{kind: layoutSpace})
}
