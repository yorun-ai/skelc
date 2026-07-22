package source

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
)

func goDocLines(name string, description string) []string {
	lines := common.SplitDocLines(description)
	if len(lines) == 0 {
		return nil
	}

	docLines := make([]string, 0, len(lines))
	docLines = append(docLines, fmt.Sprintf("%s %s", name, lines[0]))
	docLines = append(docLines, lines[1:]...)
	return docLines
}

func goMethodDocLines(name string, description string, example string, arguments []*MethodArgument, resultType *Type, outputDescription string, outputExample string) []string {
	docLines := goDocLines(name, common.MergeDescriptionAndExample(description, example))
	if len(docLines) == 0 {
		docLines = []string{name}
	}
	docLines[0] = common.EnsureSentence(docLines[0])

	paramLines := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if argument.Description == "" {
			continue
		}
		paramLines = append(paramLines, fmt.Sprintf("@param %s - %s", argument.Name, argument.Description))
	}
	if len(paramLines) > 0 {
		docLines = append(docLines, paramLines...)
	}

	outputLines := common.SplitDocLines(common.MergeDescriptionAndExample(outputDescription, outputExample))
	if len(outputLines) == 0 || resultType == nil {
		return docLines
	}

	docLines = append(docLines, fmt.Sprintf("@returns %s - %s", resultType.Plain, outputLines[0]))
	return docLines
}
