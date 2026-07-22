package golang

import (
	gomodule "go.yorun.ai/skelc/internal/codegen/golang/module"
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/model"
)

// DefaultVineVersion is the minimum Vine version targeted by generated Go code.
const DefaultVineVersion = gomodule.DefaultVineVersion

type Option struct {
	CompilerVersion string
	AsModule        bool
	Out             string
	Module          string
	PubOut          string
	PubModule       string
	Imports         map[string]string
	ModulePrefix    string
	VineVersion     string
}

type _GenOption struct {
	AsModule bool
	Out      string
	Module   string

	CompilerVersion string
	Imports         map[string]string
	ModulePrefix    string
	VineVersion     string

	Mode              view.Mode
	PubImportPath     string
	ExtraDependencies []string

	Domain *model.Domain
}
