package grammar

import (
	"strings"
	"testing"

	"github.com/alecthomas/participle/v2"
)

func buildSkelParserForTest(t *testing.T) *participle.Parser[SkelContent] {
	t.Helper()
	return participle.MustBuild[SkelContent](Options...)
}

func parseSkelForTest(t *testing.T, src string) *SkelContent {
	t.Helper()

	parser := buildSkelParserForTest(t)
	content, err := parser.ParseString("test.skel", strings.TrimSpace(src))
	if err != nil {
		t.Fatalf("parse skel: %v", err)
	}
	if err := content.Finalize(); err != nil {
		t.Fatalf("finalize skel: %v", err)
	}
	return content
}
