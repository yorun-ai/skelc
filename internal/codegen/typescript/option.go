package typescript

type Option struct {
	PubOnly     bool
	AsModule    bool
	Out         string
	Module      string
	Imports     map[string]string
	ModuleScope string
}
