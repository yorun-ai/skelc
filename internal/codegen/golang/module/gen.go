package module

import (
	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/util/checkutil"
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

func Generate(option Option) {
	file := new(modfile.File)
	checkutil.CheckNilError(file.AddModuleStmt(option.Module), "add go module statement failed")
	checkutil.CheckNilError(file.AddGoStmt(goVersion), "add go version statement failed")

	checkutil.CheckNilError(file.AddRequire(decimalModule, decimalVersion), "add go require %s failed", decimalModule)
	checkutil.CheckNilError(file.AddRequire(vineModule, option.VineVersion), "add go require %s failed", vineModule)
	for _, dependency := range goModDependencies(option.Imports, option.ExtraDependencies) {
		checkutil.CheckNilError(file.AddRequire(dependency.Module, dependency.Version), "add go require %s failed", dependency.Module)
	}

	content, err := file.Format()
	checkutil.CheckNilError(err, "format go.mod failed")
	codegen.NewRenderer(option.Out).Write(goModFilename, string(content))
}
