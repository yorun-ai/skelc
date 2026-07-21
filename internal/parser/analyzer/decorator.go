package analyzer

import (
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
)

type decoratorMeta struct {
	Description string
	Example     string
	HasExample  bool
	examplePos  lexer.Position
}

type decoratorContext struct {
	allowDesc    bool
	allowExample bool
	ignoreOthers bool
	requireDesc  bool
}

func parseDecoratorMeta(decorators []*grammar.Decorator, ctx decoratorContext) decoratorMeta {
	meta := decoratorMeta{}
	for _, decorator := range decorators {
		switch decorator.Name.Value {
		case "desc":
			checkutil.Check(ctx.allowDesc, "%s unexpected decorator %s", decorator.Name.Pos, "@"+decorator.Name.Value)
			checkutil.Check(meta.Description == "", "%s duplicated decorator @desc", decorator.Name.Pos)
			checkutil.CheckNotNil(decorator.Value, "%s decorator @desc requires a string argument", decorator.Name.Pos)
			meta.Description = grammar.UnquoteDescriptionString(decorator.Value.Raw)
		case "example":
			checkutil.Check(ctx.allowExample, "%s unexpected decorator %s", decorator.Name.Pos, "@"+decorator.Name.Value)
			checkutil.CheckNot(meta.HasExample, "%s duplicated decorator @example", decorator.Name.Pos)
			checkutil.Check(decorator.Value != nil && decorator.Value.Raw != "",
				"%s decorator @example requires a value", decorator.Name.Pos)
			meta.Example = decorator.Value.Raw
			meta.HasExample = true
			meta.examplePos = decorator.Name.Pos
		default:
			checkutil.Check(ctx.ignoreOthers,
				"%s unexpected decorator %s, only @desc/@example supported here", decorator.Name.Pos, "@"+decorator.Name.Value)
		}
	}
	checkutil.CheckNot(ctx.requireDesc && meta.HasExample && meta.Description == "",
		"%s decorator @example must be used with @desc", meta.examplePos)
	return meta
}

func parseAnnotations(decorators []*grammar.Decorator) decoratorMeta {
	return parseDecoratorMeta(decorators, decoratorContext{
		allowDesc:    true,
		allowExample: true,
		requireDesc:  true,
	})
}
