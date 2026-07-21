package formatter

import "strings"

const indentWidth = 4

type _ScanState struct {
	inBlockComment bool
	inTripleString bool
}

func Source(source []byte) []byte {
	normalized := strings.ReplaceAll(string(source), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	formatted := make([]string, 0, len(lines))
	indent := 0
	state := _ScanState{}
	blankPending := false
	for _, line := range lines {
		if state.inTripleString {
			formatted = appendPendingBlank(formatted, &blankPending)
			formatted = append(formatted, line)
			if strings.Contains(line, `"""`) {
				state.inTripleString = false
			}
			continue
		}
		if state.inBlockComment {
			formatted = appendPendingBlank(formatted, &blankPending)
			formatted = append(formatted, strings.TrimRight(line, " \t"))
			if strings.Contains(line, "*/") {
				state.inBlockComment = false
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(formatted) > 0 {
				blankPending = true
			}
			continue
		}

		formatted = appendPendingBlank(formatted, &blankPending)
		leadingClosers := countLeadingClosers(trimmed)
		lineIndent := max(0, indent-leadingClosers)
		formatted = append(formatted, strings.Repeat(" ", lineIndent*indentWidth)+trimmed)

		opens, closes := scanStructuralBraces(trimmed, &state)
		indent = max(0, indent+opens-closes)
	}

	return []byte(strings.Join(formatted, "\n") + "\n")
}

func appendPendingBlank(lines []string, pending *bool) []string {
	if *pending && len(lines) > 0 && lines[len(lines)-1] != "" {
		lines = append(lines, "")
	}
	*pending = false
	return lines
}

func countLeadingClosers(line string) int {
	count := 0
	for count < len(line) && line[count] == '}' {
		count++
	}
	return count
}

func scanStructuralBraces(line string, state *_ScanState) (int, int) {
	opens := 0
	closes := 0
	inString := false
	escaped := false
	for index := 0; index < len(line); index++ {
		if state.inBlockComment {
			if index+1 < len(line) && line[index:index+2] == "*/" {
				state.inBlockComment = false
				index++
			}
			continue
		}
		if state.inTripleString {
			if index+2 < len(line) && line[index:index+3] == `"""` {
				state.inTripleString = false
				index += 2
			}
			continue
		}
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if line[index] == '\\' {
				escaped = true
				continue
			}
			if line[index] == '"' {
				inString = false
			}
			continue
		}
		if index+1 < len(line) && line[index:index+2] == "//" {
			break
		}
		if index+1 < len(line) && line[index:index+2] == "/*" {
			state.inBlockComment = true
			index++
			continue
		}
		if index+2 < len(line) && line[index:index+3] == `"""` {
			state.inTripleString = true
			index += 2
			continue
		}
		switch line[index] {
		case '"':
			inString = true
		case '{':
			opens++
		case '}':
			closes++
		}
	}
	return opens, closes
}
