package schema

type _ActorSchema struct {
	Name           string
	SkelName       string
	Description    string
	Hash           string
	Vias           []_ActorVia
	AuthEnabled    bool
	AuthCredential *_DataSchema
	AuthInfo       *_DataSchema
	AuthService    *_ServiceSchema
	AuthMethod     *_MethodSchema
	PermEnabled    bool
	PermService    *_ServiceSchema
	PermMethod     *_MethodSchema
}

type _ActorAudienceSchema struct {
	Name     string
	SkelName string
	Via      _ActorVia
}

type _ActorVia string

const (
	actorViaClient  _ActorVia = "client"
	actorViaAgent   _ActorVia = "agent"
	actorViaOpenAPI _ActorVia = "openapi"
)

type _WebSchema struct {
	Name        string
	SkelName    string
	Description string
	Hash        string
	Audiences   []*_ActorAudienceSchema
}
