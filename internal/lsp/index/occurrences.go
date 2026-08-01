package index

import (
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
			if domain := document.Imports[tokens[index-2].Value]; domain != "" {
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
