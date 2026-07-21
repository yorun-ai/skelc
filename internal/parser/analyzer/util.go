package analyzer

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/nameutil"
)

type caseType string

const (
	caseTypeSnake          caseType = "snake_case"
	caseTypeScreamingSnake caseType = "SCREAMING_SNAKE_CASE"
	caseTypeCamel          caseType = "CamelCase"
	caseTypeLowerCamel     caseType = "lowerCamelCase"
)

var reservedKindSuffixes = []string{"Config", "Event", "Actor", "Service", "Web"}

func checkCase(kindName string, expectedCase caseType, ident *grammar.Identifier) {
	checkCaseAdvanced(kindName, "", "", expectedCase, ident)
}

func checkNotReservedKindSuffix(kindName string, ident *grammar.Identifier) {
	for _, suffix := range reservedKindSuffixes {
		checkutil.CheckNot(strings.HasSuffix(ident.Value, suffix),
			"%s %s name must not end with %s", ident.Pos, kindName, suffix)
	}
}

func checkCaseAdvanced(kindName string, prefix string, suffix string, expectedCase caseType, ident *grammar.Identifier) {
	name := ident.Value
	pos := ident.Pos

	checkutil.CheckNot(strings.HasPrefix(name, "_"), "%s unexpected leading underscore for %s: %s ", pos, kindName, name)

	expectedFormat := string(expectedCase)
	if prefix != "" {
		expectedFormat = fmt.Sprintf("[%s]%s", prefix, expectedFormat)
	}
	if suffix != "" {
		expectedFormat = fmt.Sprintf("%s[%s]", expectedFormat, suffix)
	}

	checkutil.Check(prefix == "" || strings.HasPrefix(name, prefix),
		"%s missing prefix: found=%s, expected=%s... (%s -> %s)", pos, name, prefix, kindName, expectedFormat)
	checkutil.Check(suffix == "" || strings.HasSuffix(ident.Value, suffix),
		"%s missing suffix: found=%s, expected=...%s (%s -> %s)", pos, name, suffix, kindName, expectedFormat)

	body := strings.TrimPrefix(name, prefix)
	body = strings.TrimSuffix(body, suffix)
	checkutil.Check(body != "", "%s missing body after trimming prefix & suffix: found=%s", pos, name)
	checkutil.CheckFunc(matchesCase(body, expectedCase), func() string {
		expectedName := fmt.Sprintf("%s%s%s", prefix, caseTypeExample(expectedCase), suffix)
		return fmt.Sprintf("%s incorrect case: found=%s, expected=%s (%s -> %s)", pos, name, expectedName, kindName, expectedFormat)
	})
}

func matchesCase(value string, expected caseType) bool {
	switch expected {
	case caseTypeSnake:
		return nameutil.IsSnakeCase(value)
	case caseTypeScreamingSnake:
		return nameutil.IsScreamingSnakeCase(value)
	case caseTypeCamel:
		return nameutil.IsCamelCase(value)
	case caseTypeLowerCamel:
		return nameutil.IsLowerCamelCase(value)
	default:
		return false
	}
}

func caseTypeExample(expected caseType) string {
	switch expected {
	case caseTypeSnake:
		return "snake_case"
	case caseTypeScreamingSnake:
		return "SCREAMING_SNAKE_CASE"
	case caseTypeCamel:
		return "CamelCase"
	case caseTypeLowerCamel:
		return "lowerCamelCase"
	default:
		return string(expected)
	}
}
