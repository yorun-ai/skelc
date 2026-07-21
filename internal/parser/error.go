package parser

import (
	"errors"

	"go.yorun.ai/skelc/internal/parser/analyzer"
)

func IsMissingImportError(err error) bool {
	target := new(analyzer.MissingImportError)
	return errors.As(err, &target)
}
