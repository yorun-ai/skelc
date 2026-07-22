package parser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yorun.ai/skelc/internal/formatter"
	"go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/internal/parser/grammar"
)

var benchmarkSource = []byte(`domain benchmark.demo
pub data Page<T> {
    items: list<T>
    cursor: string?
}
pub data User {
    id: uuid
    friends: list<User?>
}
resource UserResource {
    action read
}
service UserService {
    for ClientActor
    method listUsers {
        require any(UserResource:read, all(UserResource:read))
        input {
            cursor: string?
        }
        output Page<User>
    }
}
`)

func BenchmarkLexer(b *testing.B) {
	for range b.N {
		lex, err := grammar.LexerDefinition().Lex("benchmark.skel", strings.NewReader(string(benchmarkSource)))
		if err != nil {
			b.Fatal(err)
		}
		for {
			token, err := lex.Next()
			if err != nil {
				b.Fatal(err)
			}
			if token.EOF() {
				break
			}
		}
	}
}

func BenchmarkParseSource(b *testing.B) {
	for range b.N {
		if _, err := parser.ParseSource("benchmark.skel", benchmarkSource); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheck(b *testing.B) {
	directory := writeBenchmarkDirectory(b, 40)
	b.ResetTimer()
	for range b.N {
		result, err := parser.Check(parser.Option{SkelIn: directory})
		if err != nil {
			b.Fatal(err)
		}
		if len(result.Diagnostics) != 0 {
			b.Fatal(result.Diagnostics)
		}
	}
}

func BenchmarkParseDirectory(b *testing.B) {
	directory := writeBenchmarkDirectory(b, 40)
	b.ResetTimer()
	for range b.N {
		if _, err := parser.Parse(parser.Option{SkelIn: directory}); err != nil {
			b.Fatal(err)
		}
	}
}

func writeBenchmarkDirectory(b *testing.B, count int) string {
	directory := b.TempDir()
	for index := range count {
		content := fmt.Sprintf("domain benchmark.check\npub data Value%d { id: string }\n", index)
		if err := os.WriteFile(filepath.Join(directory, fmt.Sprintf("value_%d.skel", index)), []byte(content), 0o600); err != nil {
			b.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "domain.skel"), []byte("domain benchmark.check\n"), 0o600); err != nil {
		b.Fatal(err)
	}
	return directory
}

func BenchmarkWorkspaceAnalysis(b *testing.B) {
	sources := benchmarkWorkspaceSources(40)
	b.ResetTimer()
	for range b.N {
		if diagnostics := parser.AnalyzeWorkspace(sources); len(diagnostics) != 0 {
			b.Fatal(diagnostics)
		}
	}
}

func BenchmarkIncrementalWorkspaceAnalysis(b *testing.B) {
	sources := benchmarkWorkspaceSources(40)
	analyzer := parser.NewWorkspaceAnalyzer()
	if diagnostics := analyzer.Analyze(sources); len(diagnostics) != 0 {
		b.Fatal(diagnostics)
	}
	b.ResetTimer()
	for range b.N {
		if diagnostics := analyzer.Analyze(sources); len(diagnostics) != 0 {
			b.Fatal(diagnostics)
		}
	}
}

func benchmarkWorkspaceSources(count int) []parser.Source {
	sources := make([]parser.Source, 0, count)
	for index := range count {
		name := fmt.Sprintf("benchmark.d%d", index)
		content := "domain " + name + "\npub data Value { id: string }\n"
		if index > 0 {
			previous := fmt.Sprintf("benchmark.d%d", index-1)
			content = "domain " + name + "\nimport " + previous + "\npub data Value { previous: d" + fmt.Sprint(index-1) + ".Value }\n"
		}
		sources = append(sources, parser.Source{Path: fmt.Sprintf("/benchmark/%d.skel", index), Content: []byte(content)})
	}
	return sources
}

func FuzzParserFormatterAnalyzer(f *testing.F) {
	for _, seed := range [][]byte{
		benchmarkSource,
		[]byte("domain fuzz\ndata Node { next: Node? }\n"),
		[]byte("domain fuzz\ndata Box<T> { value: list<map<string, list<T?>>> }\n"),
		[]byte("domain fuzz\nresource File { action read }\nservice Files { method get { require any(File:read) } }\n"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source []byte) {
		formatted := formatter.Source(source)
		if second := formatter.Source(formatted); string(second) != string(formatted) {
			t.Fatalf("formatter is not idempotent: first=%q second=%q", formatted, second)
		}
		_, originalErr := parser.ParseSource("fuzz.skel", source)
		if originalErr == nil {
			if _, formattedErr := parser.ParseSource("fuzz.skel", formatted); formattedErr != nil {
				t.Fatalf("formatter changed valid syntax: %v", formattedErr)
			}
		}
		_ = parser.AnalyzeWorkspace([]parser.Source{{Path: "/fuzz/input.skel", Content: source}})
	})
}
