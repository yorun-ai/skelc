package analyzer

import (
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/model"
)

func parseWeb(reporter *diagnosticReporter, gw *grammar.Web, pub bool) (*model.Web, bool) {
	valid := checkCaseAdvanced(reporter, "Web", "", "Web", caseTypeCamel, gw.Name)
	meta, metaValid := parseDecoratorMeta(reporter, gw.Decorators, decoratorContext{
		allowDesc: true,
	})
	valid = metaValid && valid
	valid = reporter.checkNot(meta.HasExample, "%s web does not support decorator @example", gw.Name.Pos) && valid
	valid = reporter.checkNot(pub, "%s web %s does not support pub", gw.Name.Pos, gw.Name.Value) && valid
	audiences, audiencesValid := parseWebAudiences(reporter, gw.Audiences)
	valid = audiencesValid && valid
	valid = reporter.check(len(audiences) > 0, "%s web %s must declare at least one actor", gw.Name.Pos, gw.Name.Value) && valid
	return &model.Web{
		Pos:         position(gw.Name.Pos),
		Name:        gw.Name.Value,
		SkelName:    "",
		Description: meta.Description,
		Audiences:   audiences,
	}, valid
}

func parseWebAudiences(reporter *diagnosticReporter, audiences []*grammar.WebAudience) ([]*model.ActorAudience, bool) {
	serviceAudiences := make([]*grammar.ServiceAudience, 0, len(audiences))
	for _, audience := range audiences {
		serviceAudiences = append(serviceAudiences, &grammar.ServiceAudience{
			Pos:     audience.Pos,
			Keyword: audience.Keyword,
			Actor:   audience.Actor,
			Via:     audience.Via,
		})
	}
	return parseServiceAudiences(reporter, serviceAudiences)
}
