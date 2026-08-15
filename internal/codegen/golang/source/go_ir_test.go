package source

import (
	"strings"
	"testing"
)

func TestGoIRTemplateRendersStructuredControlFlow(t *testing.T) {
	function := goFunction(
		[]*_GoParameter{goParameter("value", "[]string")},
		"[]string",
		goBlock(
			goAssignmentStatement("cloned", ":=", goRaw("value")),
			goIfStatement(
				nil,
				goRaw("value != nil"),
				goBlock(
					goAssignmentStatement("cloned", "=", goCall("make", goRaw("[]string"), goCall("len", goRaw("value")))),
					goRangeStatement(
						[]string{"index"},
						goRaw("value"),
						goBlock(goAssignmentStatement("cloned[index]", "=", goRaw("value[index]"))),
					),
				),
				nil,
			),
			goReturnStatement(goRaw("cloned")),
		),
	)

	got := renderGoIRForTest(t, "goFunction", function)
	for _, fragment := range []string{
		"func(value []string) []string",
		"cloned := value",
		"if value != nil",
		"cloned = make([]string, len(value))",
		"for index := range value",
		"cloned[index] = value[index]",
		"return cloned",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("rendered Go IR missing %q:\n%s", fragment, got)
		}
	}
}

func TestGoIRTemplateRendersVariableAndMultiAssignment(t *testing.T) {
	block := goBlock(
		goVariableStatement("clone", "func(string) string"),
		&_GoStatement{Assignment: goAssignments(
			[]string{"value", "ok"},
			":=",
			goRaw("input.(string)"),
		)},
	)

	got := renderGoIRForTest(t, "goBlock", block)
	for _, fragment := range []string{
		"var clone func(string) string",
		"value, ok := input.(string)",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("rendered Go IR missing %q:\n%s", fragment, got)
		}
	}
}
