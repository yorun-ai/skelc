package hasher

import "go.yorun.ai/skelc/model"

type _hashState struct {
	domain      *model.Domain
	enumBySkel  map[string]*model.Enum
	dataBySkel  map[string]*model.Data
	actorBySkel map[string]*model.Actor
	status      map[string]bool
	cache       map[string]string
}

func newHashState(domain *model.Domain) *_hashState {
	state := &_hashState{
		domain:      domain,
		enumBySkel:  map[string]*model.Enum{},
		dataBySkel:  map[string]*model.Data{},
		actorBySkel: map[string]*model.Actor{},
		status:      map[string]bool{},
		cache:       map[string]string{},
	}
	for _, enum := range domain.Enums() {
		state.enumBySkel[enum.SkelName] = enum
	}
	for _, data := range domain.Data() {
		state.dataBySkel[data.SkelName] = data
	}
	for _, config := range domain.Configs() {
		state.dataBySkel[config.SkelName] = config
	}
	for _, event := range domain.Events() {
		state.dataBySkel[event.SkelName] = event
	}
	for _, actor := range domain.Actors() {
		state.actorBySkel[actor.SkelName] = actor
	}
	return state
}

func (s *_hashState) enumHash(enum *model.Enum) string {
	return s.memoHash("enum", enum.SkelName, func() string {
		return hashValue(_EnumHashValue{
			Name:        enum.Name,
			SkelName:    enum.SkelName,
			Description: enum.Description,
			Items:       buildEnumItemHashValues(enum.Items),
		})
	})
}

func (s *_hashState) dataHash(data *model.Data) string {
	return s.memoHash(string(data.Kind), data.SkelName, func() string {
		return hashValue(_DataHashValue{
			Name:           data.Name,
			SkelName:       data.SkelName,
			Description:    data.Description,
			Kind:           data.Kind,
			Pub:            data.Pub,
			Lifecycle:      string(data.Lifecycle),
			TypeParameters: buildTypeParameterNames(data.TypeParameters),
			Members:        s.buildMemberHashValues(data.Members),
		})
	})
}

func (s *_hashState) webHash(web *model.Web) string {
	return s.memoHash("web", web.SkelName, func() string {
		return hashValue(_WebHashValue{
			Name:        web.Name,
			SkelName:    web.SkelName,
			Description: web.Description,
			Actors:      s.buildActorAudienceHashValues(web.Audiences),
		})
	})
}

func (s *_hashState) actorHash(actor *model.Actor) string {
	return s.memoHash("actor", actor.SkelName, func() string {
		var authCredentialName string
		var authCredentialHash string
		var authInfoName string
		var authInfoHash string
		var authMethodName string
		var authMethodHash string
		if actor.AuthEnabled {
			authCredentialName = actor.AuthCredential.SkelName
			authCredentialHash = s.dataHash(actor.AuthCredential)
			authInfoName = actor.AuthInfo.SkelName
			authInfoHash = s.dataHash(actor.AuthInfo)
			authMethodName = actor.AuthMethod.SkelName
			authMethodHash = s.methodHash(actor.AuthMethod)
		}
		var permMethodName string
		var permMethodHash string
		if actor.PermMethod != nil {
			permMethodName = actor.PermMethod.SkelName
			permMethodHash = s.methodHash(actor.PermMethod)
		}
		return hashValue(_ActorHashValue{
			Name:               actor.Name,
			SkelName:           actor.SkelName,
			Description:        actor.Description,
			Vias:               buildActorViaNames(actor.Vias),
			AuthEnabled:        actor.AuthEnabled,
			AuthCredential:     authCredentialName,
			AuthCredentialHash: authCredentialHash,
			AuthInfo:           authInfoName,
			AuthInfoHash:       authInfoHash,
			AuthMethod:         authMethodName,
			AuthMethodHash:     authMethodHash,
			PermEnabled:        actor.PermEnabled,
			PermMethod:         permMethodName,
			PermMethodHash:     permMethodHash,
		})
	})
}
