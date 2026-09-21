package index

import (
	"strings"
	"unicode"

	"go.yorun.ai/skelc/internal/lsp/source"
)

func indexOccurrences(document *Document, tokens []source.Token) []Occurrence {
	definitions := make(map[string]string, len(document.Definitions))
	for _, definition := range document.Definitions {
		definitions[definition.Name] = definition.Key
	}
	occurrences := make([]Occurrence, 0)
	for index, token := range tokens {
		key := ""
		if index >= 2 && tokens[index-1].Value == "." {
			start := index - 2
			for start >= 2 && tokens[start-1].Value == "." && IsIdentifier(tokens[start-2].Value) {
				start -= 2
			}
			parts := []string{}
			for i := start; i < index-1; i += 2 {
				parts = append(parts, tokens[i].Value)
			}
			if domain := document.Imports[strings.Join(parts, ".")]; domain != "" {
				key = domain + "." + token.Value
			}
		} else if definitions[token.Value] != "" {
			key = definitions[token.Value]
		} else if unicode.IsUpper(source.FirstRune(token.Value)) && document.Domain != "" {
			key = document.Domain + "." + token.Value
		}
		if key != "" {
			occurrences = append(occurrences, Occurrence{Key: key, Range: source.New(document.Source).Range(token.Start, token.End)})
		}
	}
	return occurrences
}
