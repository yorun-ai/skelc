package analyzer

import (
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

func parseWeb(gw *grammar.Web, pub bool) *model.Web {
	checkCaseAdvanced("Web", "", "Web", caseTypeCamel, gw.Name)
	meta := parseDecoratorMeta(gw.Decorators, decoratorContext{
		allowDesc: true,
	})
	checkutil.CheckNot(meta.HasExample, "%s web does not support decorator @example", gw.Name.Pos)
	checkutil.CheckNot(pub, "%s web %s does not support pub", gw.Name.Pos, gw.Name.Value)
	audiences := parseWebAudiences(gw.Audiences)
	checkutil.Check(len(audiences) > 0, "%s web %s must declare at least one actor", gw.Name.Pos, gw.Name.Value)
	return &model.Web{
		Pos:         position(gw.Name.Pos),
		Name:        gw.Name.Value,
		SkelName:    "",
		Description: meta.Description,
		Audiences:   audiences,
	}
}

func parseWebAudiences(audiences []*grammar.WebAudience) []*model.ActorAudience {
	serviceAudiences := make([]*grammar.ServiceAudience, 0, len(audiences))
	for _, audience := range audiences {
		serviceAudiences = append(serviceAudiences, &grammar.ServiceAudience{
			Pos:     audience.Pos,
			Keyword: audience.Keyword,
			Actor:   audience.Actor,
			Via:     audience.Via,
		})
	}
	return parseServiceAudiences(serviceAudiences)
}
