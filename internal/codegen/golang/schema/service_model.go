package schema

type _ServiceSchema struct {
	Name        string
	SkelName    string
	Description string
	Hash        string
	Pub         bool
	AuthMode    _AuthMode
	Audiences   []*_ActorAudienceSchema
	Require     *_PermRequire
	Methods     []*_MethodSchema
}

type _MethodSchema struct {
	Name              string
	SkelName          string
	Description       string
	Hash              string
	Example           string
	AuthMode          _AuthMode
	Require           *_PermRequire
	InputDescription  string
	OutputDescription string
	OutputExample     string
	Arguments         []*_MemberSchema
	ResultType        *_TypeSchema
}

type _AuthMode string

const (
	authModeUnset  _AuthMode = "unset"
	authModeAuth   _AuthMode = "auth"
	authModeNoAuth _AuthMode = "noauth"
)
