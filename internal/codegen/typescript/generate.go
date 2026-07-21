package typescript

import (
	"go.yorun.ai/skelc/internal/codegen/typescript/module"
	"go.yorun.ai/skelc/internal/codegen/typescript/source"
	"go.yorun.ai/skelc/model"
)

func Generate(domain *model.Domain, option Option) {
	result := source.Generate(domain, option.Out, source.Option{
		PubOnly:     option.PubOnly,
		ModuleScope: option.ModuleScope,
		Module:      option.Module,
		Imports:     option.Imports,
	})
	if option.AsModule {
		module.Generate(module.Option{
			Out:             option.Out,
			PackageName:     result.PackageName,
			Imports:         option.Imports,
			ResolvedImports: result.ResolvedImports,
		})
	}
}
