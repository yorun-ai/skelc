package module

import (
	"fmt"

	"go.yorun.ai/skelc/internal/codegen/common"
	"golang.org/x/mod/modfile"
)

const goModFilename = "go.mod"

const (
	goVersion      = "1.26"
	decimalModule  = "github.com/shopspring/decimal"
	decimalVersion = "v1.4.0"
	vineModule     = "go.yorun.ai/vine"

	defaultGoImportVersion = "v0.0.0-00010101000000-000000000000"
)

type Option struct {
	Out               string
	Module            string
	VineVersion       string
	Imports           map[string]string
	ExtraDependencies []string
}

func Generate(option Option) error {
	file := new(modfile.File)
	if err := file.AddModuleStmt(option.Module); err != nil {
		return fmt.Errorf("add Go module statement: %w", err)
	}
	if err := file.AddGoStmt(goVersion); err != nil {
		return fmt.Errorf("add Go version statement: %w", err)
	}

	if err := file.AddRequire(decimalModule, decimalVersion); err != nil {
		return fmt.Errorf("add Go requirement %s: %w", decimalModule, err)
	}
	if err := file.AddRequire(vineModule, option.VineVersion); err != nil {
		return fmt.Errorf("add Go requirement %s: %w", vineModule, err)
	}
	dependencies, err := goModDependencies(option.Imports, option.ExtraDependencies)
	if err != nil {
		return err
	}
	for _, dependency := range dependencies {
		if err := file.AddRequire(dependency.Module, dependency.Version); err != nil {
			return fmt.Errorf("add Go requirement %s: %w", dependency.Module, err)
		}
	}

	content, err := file.Format()
	if err != nil {
		return fmt.Errorf("format go.mod: %w", err)
	}
	renderer := common.NewRenderer(option.Out)
	renderer.Write(goModFilename, string(content))
	return renderer.Err()
}
