package analyzer

import (
	"fmt"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

func parseService(gs *grammar.Service) *model.Service {
	checkCaseAdvanced("Service", "", "Service", caseTypeCamel, gs.Name)
	meta := parseDecoratorMeta(gs.Decorators, decoratorContext{
		allowDesc: true,
	})
	checkutil.CheckNot(meta.HasExample, "%s service does not support decorator @example", gs.Name.Pos)
	audiences := parseServiceAudiences(serviceAudiences(gs))
	auth := parseAuthMode(serviceAuthMarker(gs), model.AuthModeUnset)
	require := parseRequire(serviceRequire(gs))
	methods := parseMethods(gs.Name, serviceMethods(gs))
	return &model.Service{
		Pos:         position(gs.Name.Pos),
		Name:        gs.Name.Value,
		Pub:         gs.Pub,
		Audiences:   audiences,
		Auth:        auth,
		Require:     require,
		Description: meta.Description,
		Methods:     methods,
		SkelName:    "",
	}
}

func serviceAudiences(gs *grammar.Service) []*grammar.ServiceAudience {
	if len(gs.Sections) == 0 {
		return gs.Audiences
	}
	audiences := make([]*grammar.ServiceAudience, 0)
	for _, section := range gs.Sections {
		if section.Audience != nil {
			audiences = append(audiences, section.Audience)
		}
	}
	return audiences
}

func serviceMethods(gs *grammar.Service) []*grammar.Method {
	if len(gs.Sections) == 0 {
		return gs.Methods
	}
	methods := make([]*grammar.Method, 0)
	for _, section := range gs.Sections {
		if section.Method != nil {
			section.Method.Decorators = append(section.Decorators, section.Method.Decorators...)
			methods = append(methods, section.Method)
		}
	}
	return methods
}

func serviceAuthMarker(gs *grammar.Service) *grammar.AuthMarker {
	if len(gs.Sections) == 0 {
		return gs.Auth
	}
	var marker *grammar.AuthMarker
	var markerPos lexer.Position
	for _, section := range gs.Sections {
		if section.Auth == nil {
			continue
		}
		checkutil.CheckFunc(marker == nil, func() string {
			return fmt.Sprintf("%s duplicated service auth marker found, also present at %s", section.Auth.Pos, markerPos)
		})
		marker = section.Auth
		markerPos = section.Auth.Pos
	}
	return marker
}

func serviceRequire(gs *grammar.Service) *grammar.Require {
	if len(gs.Sections) == 0 {
		return nil
	}
	var require *grammar.Require
	var requirePos lexer.Position
	for _, section := range gs.Sections {
		if section.Require == nil {
			continue
		}
		checkutil.CheckFunc(require == nil, func() string {
			return fmt.Sprintf("%s duplicated service require found, also present at %s", section.Require.Pos, requirePos)
		})
		require = section.Require
		requirePos = section.Require.Pos
	}
	return require
}

func parseAuthMode(marker *grammar.AuthMarker, defaultMode model.AuthMode) model.AuthMode {
	if marker == nil {
		return defaultMode
	}
	switch marker.Value {
	case string(model.AuthModeAuth):
		return model.AuthModeAuth
	case string(model.AuthModeNoAuth):
		return model.AuthModeNoAuth
	}
	panic("unexpected auth marker " + marker.Value)
}

func parseServiceAudiences(audiences []*grammar.ServiceAudience) []*model.ActorAudience {
	if len(audiences) == 0 {
		return []*model.ActorAudience{}
	}
	parsed := make([]*model.ActorAudience, 0, len(audiences))
	audiencePos := map[string]lexer.Position{}
	for _, audience := range audiences {
		actorIdent := audience.Actor.Parts[len(audience.Actor.Parts)-1]
		checkCaseAdvanced("Actor", "", "Actor", caseTypeCamel, actorIdent)
		via := ""
		if audience.Via != nil {
			via = audience.Via.Value
		}
		name := audience.Actor.String()
		key := fmt.Sprintf("%s:%s", name, via)
		duplicatedPosition, duplicated := audiencePos[key]
		checkutil.CheckFunc(!duplicated, func() string {
			if via != "" {
				return fmt.Sprintf("%s duplicated service audience %s via %s found, also present at %s", actorIdent.Pos, name, via, duplicatedPosition)
			}
			return fmt.Sprintf("%s duplicated service audience %s found, also present at %s", actorIdent.Pos, name, duplicatedPosition)
		})
		audiencePos[key] = actorIdent.Pos
		parsed = append(parsed, &model.ActorAudience{
			Actor: name,
			Via:   via,
			Pos:   position(audience.Pos),
		})
	}
	return parsed
}
