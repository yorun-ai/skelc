package cli

import (
	"errors"
	"fmt"
	"path/filepath"

	"go.yorun.ai/skelc/internal/parser"
)

type _OptionValidationError interface {
	ValidationField() string
	ValidationRule() string
}

func formatGenerationError(err error) error {
	var validationError _OptionValidationError
	if !errors.As(err, &validationError) {
		return err
	}
	message := map[string]string{
		"skel.input\x00required":                     "missing flag skel-in",
		"go.output\x00required":                      "missing output flag",
		"go.module-identity\x00required":             "missing flag go-module or go-module-prefix",
		"go.public-output\x00requires-module":        "flag go-pub-out requires go-module output",
		"go.public-module\x00requires-module":        "flag go-pub-module requires go-module output",
		"go.module\x00requires-module":               "flag go-module requires go-module output",
		"go.module-prefix\x00requires-module":        "flag go-module-prefix requires go-module output",
		"go.public-module\x00requires-public-output": "flag go-pub-module requires go-pub-out",
		"go.module-prefix\x00no-trailing-slash":      "flag go-module-prefix must not end with /",
		"go.module\x00no-trailing-slash":             "flag go-module must not end with /",
		"go.public-module\x00no-trailing-slash":      "flag go-pub-module must not end with /",
		"go.import\x00no-trailing-slash":             "flag go-import must not end with /",
		"typescript.output\x00required":              "missing output flag",
		"typescript.module-identity\x00required":     "missing flag ts-module or ts-module-scope",
		"typescript.module\x00requires-module":       "flag ts-module requires ts-as-module",
		"typescript.module-scope\x00requires-module": "flag ts-module-scope requires ts-as-module",
		"skeleton.output\x00required":                "missing output flag",
		"skeleton.public-only\x00required":           "flag pub is required for gen skel",
	}[validationError.ValidationField()+"\x00"+validationError.ValidationRule()]
	if message == "" {
		return err
	}
	return errors.New(message)
}

func normalizeParserOption(option *parser.Option) error {
	if option.SkelIn == "" {
		return fmt.Errorf("missing flag skel-in")
	}
	path, err := filepath.Abs(option.SkelIn)
	if err != nil {
		return fmt.Errorf("resolve path %s: %w", option.SkelIn, err)
	}
	option.SkelIn = path
	return nil
}
