package source

import (
	"fmt"

	"go.yorun.ai/skelc/model"
)

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
	types        []*model.Type
	err          error
}

func (b *_WireSchemaBuilder) fail(format string, args ...any) {
	if b.err == nil {
		b.err = fmt.Errorf(format, args...)
	}
}

func newWireSchemaBuilder() *_WireSchemaBuilder {
	return &_WireSchemaBuilder{
		data:         map[*model.Data]bool{},
		factoryNames: map[*model.Data]string{},
		types:        make([]*model.Type, 0),
	}
}
