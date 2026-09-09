package source

import "go.yorun.ai/skelc/internal/model"

func methodArgumentsContainBinary(method *model.Method) bool {
	for _, argument := range method.Arguments {
		if argument.Type.ContainsBinaryType() {
			return true
		}
	}
	return false
}

func methodResultContainsBinary(method *model.Method) bool {
	return method.ResultType.ContainsBinaryType()
}
