package source

import "go.yorun.ai/skelc/model"

type _WireMethod struct {
	Name            string
	ArgumentsSchema string
	ResultSchema    string
}

type _WireFactory struct {
	Code string
}

type _WireSchemaBuilder struct {
	data         map[*model.Data]bool
	factoryNames map[*model.Data]string
}

func newWireSchemaBuilder() *_WireSchemaBuilder {
	return &_WireSchemaBuilder{
		data:         map[*model.Data]bool{},
		factoryNames: map[*model.Data]string{},
	}
}
