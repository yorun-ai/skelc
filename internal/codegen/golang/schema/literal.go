package schema

import (
	"strconv"

	"go.yorun.ai/skelc/model"
)

func renderScalarLiteral(scalar _Scalar) string {
	switch scalar {
	case scalarString:
		return "skel.ScalarString"
	case scalarBool:
		return "skel.ScalarBool"
	case scalarInt:
		return "skel.ScalarInt"
	case scalarLong:
		return "skel.ScalarLong"
	case scalarFloat:
		return "skel.ScalarFloat"
	case scalarDouble:
		return "skel.ScalarDouble"
	case scalarDecimal:
		return "skel.ScalarDecimal"
	case scalarJson:
		return "skel.ScalarJson"
	case scalarUuid:
		return "skel.ScalarUuid"
	case scalarTimestamp:
		return "skel.ScalarTimestamp"
	case scalarDuration:
		return "skel.ScalarDuration"
	case scalarLocalDate:
		return "skel.ScalarLocalDate"
	case scalarLocalTime:
		return "skel.ScalarLocalTime"
	case scalarLocalDateTime:
		return "skel.ScalarLocalDateTime"
	case scalarBinary:
		return "skel.ScalarBinary"
	default:
		return "skel.Scalar(" + quote(string(scalar)) + ")"
	}
}

func actorVia(name string) _ActorVia {
	switch model.ActorViaKind(name) {
	case model.ActorViaClient:
		return actorViaClient
	case model.ActorViaAgent:
		return actorViaAgent
	case model.ActorViaOpenAPI:
		return actorViaOpenAPI
	default:
		panic("unexpected actor via " + name)
	}
}

func renderActorViaLiteral(method _ActorVia) string {
	switch method {
	case actorViaClient:
		return "skel.ActorViaClient"
	case actorViaAgent:
		return "skel.ActorViaAgent"
	case actorViaOpenAPI:
		return "skel.ActorViaOpenAPI"
	default:
		panic("unexpected actor via")
	}
}

func renderAuthModeLiteral(mode _AuthMode) string {
	switch mode {
	case authModeUnset:
		return "skel.AuthModeUnset"
	case authModeAuth:
		return "skel.AuthModeAuth"
	case authModeNoAuth:
		return "skel.AuthModeNoAuth"
	default:
		panic("unexpected auth mode")
	}
}

func renderPermRequireModeLiteral(mode _PermRequireMode) string {
	switch mode {
	case permRequireModeCode:
		return "skel.PermRequireModeCode"
	case permRequireModeCheck:
		return "skel.PermRequireModeCheck"
	case permRequireModeAll:
		return "skel.PermRequireModeAll"
	case permRequireModeAny:
		return "skel.PermRequireModeAny"
	default:
		panic("unexpected permission require mode")
	}
}

func quote(value string) string {
	return strconv.Quote(value)
}
