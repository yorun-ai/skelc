package typescript

type Option struct {
	AsModule    bool
	Out         string
	Module      string
	Imports     map[string]string
	ModuleScope string
}
