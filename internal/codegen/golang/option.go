package golang

import (
	gomodule "go.yorun.ai/skelc/internal/codegen/golang/module"
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/internal/model"
)

// MinimumVineVersion is the minimum Vine version supported by generated Go code.
const MinimumVineVersion = gomodule.MinimumVineVersion

// DefaultVineVersion is the Vine version used when generation does not select one.
const DefaultVineVersion = gomodule.DefaultVineVersion

// MinimumApiServiceVineVersion is required by schemas for explicit API services.
const MinimumApiServiceVineVersion = gomodule.MinimumApiServiceVineVersion

type Option struct {
	CompilerVersion string
	AsModule        bool
	PubOnly         bool
	ApiOnly         bool
	Out             string
	Module          string
	PubOut          string
	PubModule       string
	Imports         map[string]string
	ModulePrefix    string
	VineVersion     string
	VrpcVersion     string
}

type _GenOption struct {
	AsModule bool
	Out      string
	Module   string

	CompilerVersion string
	Imports         map[string]string
	ModulePrefix    string
	VineVersion     string
	VrpcVersion     string

	Mode              view.Mode
	PubImportPath     string
	ExtraDependencies []string

	Domain *model.Domain
}
