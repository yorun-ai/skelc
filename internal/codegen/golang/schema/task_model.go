package schema

type _TaskSchema struct {
	Name        string
	SkelName    string
	Description string
	Hash        string
	Triggers    []*_TriggerSchema
}

type _TriggerSchema struct {
	Name             string
	SkelName         string
	Description      string
	Hash             string
	Example          string
	InputDescription string
	Arguments        []*_MemberSchema
}
