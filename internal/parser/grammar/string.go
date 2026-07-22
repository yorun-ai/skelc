package grammar

import (
	"fmt"
	"strconv"
	"strings"
)

func UnquoteDescriptionString(raw string) (string, error) {
	if strings.HasPrefix(raw, `"""`) && strings.HasSuffix(raw, `"""`) && len(raw) >= 6 {
		return unquoteTripleQuotedDescription(raw)
	}
	return UnquoteDoubleQuotedString(raw)
}

func unquoteTripleQuotedDescription(raw string) (string, error) {
	content := raw[3 : len(raw)-3]
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) < 2 || strings.TrimSpace(lines[0]) != "" || strings.TrimSpace(lines[len(lines)-1]) != "" {
		return "", fmt.Errorf("expected triple-quoted string to use standalone delimiter lines")
	}
	return dedentTripleQuotedDescription(strings.Join(lines[1:len(lines)-1], "\n")), nil
}

func dedentTripleQuotedDescription(content string) string {
	lines := strings.Split(content, "\n")
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := leadingWhitespaceLen(line)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return content
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines[i] = line[minIndent:]
	}
	return strings.Join(lines, "\n")
}

func leadingWhitespaceLen(line string) int {
	for index, char := range line {
		if char != ' ' && char != '\t' {
			return index
		}
	}
	return len(line)
}

func UnquoteDoubleQuotedString(raw string) (string, error) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", fmt.Errorf("expected double-quoted string literal")
	}

	var output strings.Builder
	content := raw[1 : len(raw)-1]
	for len(content) > 0 {
		if content[0] != '\\' {
			output.WriteByte(content[0])
			content = content[1:]
			continue
		}

		unquoted, multiByte, tail, err := strconv.UnquoteChar(content, '"')
		if err != nil {
			return "", fmt.Errorf("unquote char: %w", err)
		}
		if multiByte {
			output.WriteRune(unquoted)
		} else {
			output.WriteByte(byte(unquoted))
		}
		content = tail
	}

	return output.String(), nil
}
