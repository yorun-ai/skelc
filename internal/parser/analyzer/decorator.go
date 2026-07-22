package analyzer

import (
	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
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

func parseDecoratorMeta(reporter *diagnosticReporter, decorators []*grammar.Decorator, ctx decoratorContext) (decoratorMeta, bool) {
	meta := decoratorMeta{}
	valid := true
	for _, decorator := range decorators {
		switch decorator.Name.Value {
		case "desc":
			accepted := reporter.check(ctx.allowDesc, "%s unexpected decorator %s", decorator.Name.Pos, "@"+decorator.Name.Value)
			accepted = reporter.check(meta.Description == "", "%s duplicated decorator @desc", decorator.Name.Pos) && accepted
			accepted = reporter.check(decorator.Value != nil, "%s decorator @desc requires a string argument", decorator.Name.Pos) && accepted
			valid = accepted && valid
			if !accepted {
				continue
			}
			description, err := grammar.UnquoteDescriptionString(decorator.Value.Raw)
			if err != nil {
				reporter.reportf("%s invalid decorator @desc: %v", decorator.Name.Pos, err)
				valid = false
				continue
			}
			meta.Description = description
		case "example":
			accepted := reporter.check(ctx.allowExample, "%s unexpected decorator %s", decorator.Name.Pos, "@"+decorator.Name.Value)
			accepted = reporter.checkNot(meta.HasExample, "%s duplicated decorator @example", decorator.Name.Pos) && accepted
			accepted = reporter.check(decorator.Value != nil && decorator.Value.Raw != "",
				"%s decorator @example requires a value", decorator.Name.Pos) && accepted
			valid = accepted && valid
			if !accepted {
				continue
			}
			meta.Example = decorator.Value.Raw
			meta.HasExample = true
			meta.examplePos = decorator.Name.Pos
		default:
			valid = reporter.check(ctx.ignoreOthers,
				"%s unexpected decorator %s, only @desc/@example supported here", decorator.Name.Pos, "@"+decorator.Name.Value) && valid
		}
	}
	valid = reporter.checkNot(ctx.requireDesc && meta.HasExample && meta.Description == "",
		"%s decorator @example must be used with @desc", meta.examplePos) && valid
	return meta, valid
}

func parseAnnotations(reporter *diagnosticReporter, decorators []*grammar.Decorator) (decoratorMeta, bool) {
	return parseDecoratorMeta(reporter, decorators, decoratorContext{
		allowDesc:    true,
		allowExample: true,
		requireDesc:  true,
	})
}
