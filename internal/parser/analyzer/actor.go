package analyzer

import (
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/parser/grammar"
	"go.yorun.ai/skelc/internal/util/sliceutil"
	"go.yorun.ai/skelc/model"
)

var actorViaKinds = []model.ActorViaKind{
	model.ActorViaClient,
	model.ActorViaAgent,
	model.ActorViaOpenAPI,
}

type _ActorAuth struct {
	Credential *model.Data
	Info       *model.Data
}

func parseActor(reporter *diagnosticReporter, ga *grammar.Actor) (*model.Actor, bool) {
	valid := checkCaseAdvanced(reporter, "Actor", "", "Actor", caseTypeCamel, ga.Name)
	meta, metaValid := parseDecoratorMeta(reporter, ga.Decorators, decoratorContext{
		allowDesc: true,
	})
	valid = metaValid && valid
	valid = reporter.checkNot(meta.HasExample, "%s actor does not support decorator @example", ga.Name.Pos) && valid
	vias, viasValid := parseActorVias(reporter, ga.Name, ga.Vias)
	valid = viasValid && valid
	auth, authValid := parseActorAuth(reporter, ga)
	valid = authValid && valid
	var authCredential *model.Data
	var authInfo *model.Data
	if auth != nil {
		authCredential = auth.Credential
		authInfo = auth.Info
	}
	permEnabled, permissionValid := actorPermissionDeclared(reporter, ga)
	valid = permissionValid && valid
	return &model.Actor{
		Pos:            position(ga.Name.Pos),
		Name:           ga.Name.Value,
		SkelName:       "",
		Description:    meta.Description,
		Pub:            ga.Pub,
		Vias:           vias,
		AuthEnabled:    auth != nil,
		AuthCredential: authCredential,
		AuthInfo:       authInfo,
		PermEnabled:    permEnabled,
	}, valid
}

func parseActorAuth(reporter *diagnosticReporter, ga *grammar.Actor) (*_ActorAuth, bool) {
	authSection, valid := actorAuthSection(reporter, ga)
	if authSection == nil {
		return nil, valid
	}
	credential, credentialValid := parseActorCredential(reporter, ga, authSection)
	info, infoValid := parseActorInfo(reporter, ga, authSection)
	return &_ActorAuth{Credential: credential, Info: info}, credentialValid && infoValid && valid
}

func parseActorCredential(reporter *diagnosticReporter, ga *grammar.Actor, authSection *grammar.ActorAuth) (*model.Data, bool) {
	credentialSection := authSection.Credential
	name := &grammar.Identifier{
		Pos:   ga.Name.Pos,
		Value: ga.Name.Value + "Credential",
	}
	credential, valid := parseDataLike(reporter, &grammar.Data{
		Pos:        credentialSection.Pos,
		Pub:        ga.Pub,
		Name:       name,
		Members:    credentialSection.Members,
		Decorators: []*grammar.Decorator{},
	}, model.DataKindData)
	valid = reporter.check(len(credential.Members) > 0, "%s actor credential must have at least one member", credentialSection.Pos) && valid
	for _, member := range credential.Members {
		valid = reporter.check(member.Type.Kind == model.TypeKindScalar && member.Type.Scalar == model.ScalarString && !member.Type.Nullable,
			"%s actor credential member %s must be string", member.Pos, member.Name) && valid
	}
	credential.Pub = ga.Pub
	return credential, valid
}

func parseActorInfo(reporter *diagnosticReporter, ga *grammar.Actor, authSection *grammar.ActorAuth) (*model.Data, bool) {
	infoSection := authSection.Info
	name := &grammar.Identifier{
		Pos:   ga.Name.Pos,
		Value: ga.Name.Value + "Info",
	}
	info, valid := parseDataLike(reporter, &grammar.Data{
		Pos:        infoSection.Pos,
		Pub:        ga.Pub,
		Name:       name,
		Members:    infoSection.Members,
		Decorators: []*grammar.Decorator{},
	}, model.DataKindData)
	info.Pub = ga.Pub
	return info, valid
}

func actorAuthSection(reporter *diagnosticReporter, ga *grammar.Actor) (*grammar.ActorAuth, bool) {
	var auth *grammar.ActorAuth
	var authPos lexer.Position
	valid := true
	for _, section := range ga.Sections {
		if section.Auth == nil {
			continue
		}
		if auth != nil {
			reporter.reportf("%s duplicated actor auth found, also present at %s", section.Auth.Pos, authPos)
			valid = false
			continue
		}
		auth = section.Auth
		authPos = section.Auth.Pos
	}
	if auth == nil {
		return nil, valid
	}
	valid = reporter.check(auth.Credential != nil && auth.Info != nil,
		"%s actor %s auth must define credential and info together", ga.Name.Pos, ga.Name.Value) && valid
	if auth.Credential == nil || auth.Info == nil {
		return nil, false
	}
	return auth, valid
}

func actorPermissionDeclared(reporter *diagnosticReporter, ga *grammar.Actor) (bool, bool) {
	var permission *grammar.ActorPermission
	var permissionPos lexer.Position
	valid := true
	for _, section := range ga.Sections {
		if section.Permission == nil {
			continue
		}
		if permission != nil {
			reporter.reportf("%s duplicated actor permission found, also present at %s", section.Permission.Pos, permissionPos)
			valid = false
			continue
		}
		permission = section.Permission
		permissionPos = section.Permission.Pos
	}
	if permission == nil {
		return false, valid
	}
	return true, valid
}

func parseActorVias(reporter *diagnosticReporter, owner *grammar.Identifier, grammarVias []*grammar.ActorVia) ([]*model.ActorVia, bool) {
	valid := reporter.check(len(grammarVias) > 0, "%s actor %s must have at least one via", owner.Pos, owner.Value)

	parsedVias := make([]*model.ActorVia, 0, len(grammarVias))
	viaPos := map[string]lexer.Position{}
	for _, grammarVia := range grammarVias {
		via, viaValid := parseActorVia(reporter, grammarVia)
		valid = viaValid && valid
		duplicatedPosition, duplicated := viaPos[via.Name]
		if duplicated {
			reporter.reportf("%s duplicated actor via %s found, also present at %s", via.Pos, via.Name, duplicatedPosition)
			valid = false
			continue
		}
		viaPos[via.Name] = lexer.Position{Filename: via.Pos.File, Line: via.Pos.Line, Column: via.Pos.Column}
		parsedVias = append(parsedVias, via)
	}
	return parsedVias, valid
}

func parseActorVia(reporter *diagnosticReporter, gv *grammar.ActorVia) (*model.ActorVia, bool) {
	valid := checkCase(reporter, "ActorVia", caseTypeLowerCamel, gv.Name)
	_, ok := sliceutil.Find(actorViaKinds, func(candidate model.ActorViaKind) bool {
		return string(candidate) == gv.Name.Value
	})
	valid = reporter.check(ok, "%s unexpected actor via %s, supported=client/agent/openapi", gv.Name.Pos, gv.Name.Value) && valid
	return &model.ActorVia{
		Name: gv.Name.Value,
		Pos:  position(gv.Name.Pos),
	}, valid
}

func buildActorAuthService(actor *model.Actor) *model.Service {
	if !actor.AuthEnabled {
		return nil
	}
	serviceName := actor.Name + "AuthService"
	credentialType := dataRefType(actor.AuthCredential)
	infoType := dataRefType(actor.AuthInfo)
	credentialArgument := &model.Argument{
		Name: "credential",
		Pos:  actor.AuthCredential.Pos,
		Type: credentialType,
	}
	credentialMethod := &model.Method{
		Name:       "auth",
		SkelName:   "auth",
		Pos:        actor.Pos,
		Auth:       model.AuthModeAuth,
		Arguments:  []*model.Argument{credentialArgument},
		ResultType: infoType,
	}
	credentialMethod.ArgumentsData = &model.Data{
		Name:     serviceName + "AuthArguments",
		Domain:   actor.AuthCredential.Domain,
		SkelName: actor.AuthCredential.Domain + "." + serviceName + "AuthArguments",
		Members:  buildArgumentMembers(credentialMethod.Arguments),
	}
	actor.AuthMethod = credentialMethod
	return &model.Service{
		Name:     serviceName,
		SkelName: actor.AuthCredential.Domain + "." + serviceName,
		Pos:      actor.Pos,
		Methods:  []*model.Method{credentialMethod},
	}
}

func buildActorPermissionService(actor *model.Actor) *model.Service {
	if !actor.PermEnabled {
		return nil
	}
	serviceName := actor.Name + "PermissionService"
	domain := strings.TrimSuffix(actor.SkelName, "."+actor.Name)
	skelPrefix := domain + "."
	codesArgument := &model.Argument{
		Name: "codes",
		Pos:  actor.Pos,
		Type: &model.Type{
			Kind: model.TypeKindList,
			List: &model.ListType{Value: &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarString}},
		},
	}
	method := &model.Method{
		Name:      "checkCodes",
		SkelName:  "checkCodes",
		Pos:       actor.Pos,
		Auth:      model.AuthModeAuth,
		Arguments: []*model.Argument{codesArgument},
		ResultType: &model.Type{
			Kind: model.TypeKindMap,
			Map: &model.MapType{
				Key:   &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarString},
				Value: &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarBoolean},
			},
		},
	}
	method.ArgumentsData = &model.Data{
		Name:     serviceName + "CheckCodesArguments",
		Domain:   domain,
		SkelName: skelPrefix + serviceName + "CheckCodesArguments",
		Members:  buildArgumentMembers(method.Arguments),
	}
	actor.PermMethod = method
	return &model.Service{
		Name:     serviceName,
		SkelName: skelPrefix + serviceName,
		Pos:      actor.Pos,
		Methods:  []*model.Method{method},
	}
}

func dataRefType(data *model.Data) *model.Type {
	return &model.Type{
		Kind:     model.TypeKindData,
		Data:     data,
		SkelName: data.SkelName,
	}
}
