package schema

type _EnumSchema struct {
	Name        string
	SkelName    string
	Description string
	Hash        string
	Items       []*_EnumItemSchema
}

type _EnumItemSchema struct {
	Name        string
	Description string
}

type _DataSchema struct {
	Name           string
	SkelName       string
	Description    string
	Hash           string
	Sensitive      bool
	TypeParameters []string
	Members        []*_MemberSchema
}

type _ConfigSchema struct {
	Name        string
	SkelName    string
	Description string
	Hash        string
	Pub         bool
	Sensitive   bool
	Lifecycle   string
	Members     []*_MemberSchema
}

type _EventSchema struct {
	Name        string
	SkelName    string
	Description string
	Hash        string
	Pub         bool
	Sensitive   bool
	Members     []*_MemberSchema
}
