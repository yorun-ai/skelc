package typescript

import (
	"go.yorun.ai/skelc/internal/codegen/typescript/module"
	"go.yorun.ai/skelc/internal/codegen/typescript/source"
	"go.yorun.ai/skelc/model"
)

func Generate(domain *model.Domain, option Option) error {
	result, err := source.Generate(domain, option.Out, source.Option{
		PubOnly:     option.PubOnly,
		ModuleScope: option.ModuleScope,
		Module:      option.Module,
		Imports:     option.Imports,
	})
	if err != nil {
		return err
	}
	if option.AsModule {
		return module.Generate(module.Option{
			Out:             option.Out,
			PackageName:     result.PackageName,
			Imports:         option.Imports,
			ResolvedImports: result.ResolvedImports,
		})
	}
	return nil
}
