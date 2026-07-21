package schema

import (
	"testing"

	"go.yorun.ai/skelc/model"
)

func fillModelHashesForTest(pkg *model.Domain) {
	pkg.SetHash("domain-hash")
	for _, data := range pkg.Data() {
		data.Hash = "data-hash"
	}
	for _, actor := range pkg.Actors() {
		actor.Hash = "actor-hash"
	}
	for _, service := range pkg.Services() {
		service.Hash = "service-hash"
		for _, method := range service.Methods {
			method.Hash = "method-hash"
		}
	}
}

func buildModelDomainForTest(t *testing.T, spec model.DomainSpec) *model.Domain {
	t.Helper()
	for _, data := range spec.Data {
		data.Kind = model.DataKindData
		data.Domain = spec.Name
	}
	for _, config := range spec.Configs {
		config.Kind = model.DataKindConfig
		config.Domain = spec.Name
	}
	for _, event := range spec.Events {
		event.Kind = model.DataKindEvent
		event.Domain = spec.Name
	}
	for _, actor := range spec.Actors {
		if actor.AuthEnabled && actor.AuthService == nil {
			for _, data := range []*model.Data{actor.AuthCredential, actor.AuthInfo} {
				data.Kind = model.DataKindData
				data.Domain = spec.Name
				if data.SkelName == "" {
					data.SkelName = spec.Name + "." + data.Name
				}
			}
			method := &model.Method{
				Name:       "auth",
				SkelName:   "auth",
				Auth:       model.AuthModeNoAuth,
				ResultType: dataTypeForTest(actor.AuthInfo),
				Arguments: []*model.Argument{
					{Name: "credential", Type: dataTypeForTest(actor.AuthCredential)},
				},
			}
			actor.AuthMethod = method
			actor.AuthService = &model.Service{
				Name:     actor.Name + "AuthService",
				SkelName: spec.Name + "." + actor.Name + "AuthService",
				Auth:     model.AuthModeNoAuth,
				Methods:  []*model.Method{method},
			}
		}
	}
	return model.NewDomainFromSpec(spec)
}

func stringTypeForTest() *model.Type {
	return &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarString}
}

func intTypeForTest() *model.Type {
	return &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarInt}
}

func dataTypeForTest(data *model.Data) *model.Type {
	return &model.Type{Kind: model.TypeKindData, Data: data, SkelName: data.SkelName}
}

func actorViaForTest(kind model.ActorViaKind) *model.ActorVia {
	return &model.ActorVia{Name: string(kind)}
}
