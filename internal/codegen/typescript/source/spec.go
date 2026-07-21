package source

import _ "embed"

const specTsFilename = "spec.ts"

//go:embed tpl/spec.ts.tpl
var specTsTemplate string

type SpecTsPayload struct {
	HasWire       bool
	WireFactories []*_WireFactory
	Services      []*Service
}

func (g *_Gen) genSpecTs() {
	payload := g.buildSpecTsPayload()
	g.renderTs(specTsFilename, specTsTemplate, payload)
}

func (g *_Gen) buildSpecTsPayload() *SpecTsPayload {
	serviceTokens := g.serviceClientServices()
	services := castServices(serviceTokens)
	builder := newWireSchemaBuilder()
	for _, service := range serviceTokens {
		for _, method := range service.Methods {
			builder.collectMethod(method)
		}
	}
	builder.prepareFactoryNames()

	hasWire := false
	for serviceIndex, service := range serviceTokens {
		for _, method := range service.Methods {
			if !methodArgumentsContainBinary(method) && !methodResultContainsBinary(method) {
				continue
			}
			hasWire = true
			services[serviceIndex].WireMethods = append(
				services[serviceIndex].WireMethods,
				builder.renderMethod(method),
			)
		}
	}

	return &SpecTsPayload{
		HasWire:       hasWire,
		WireFactories: builder.renderFactories(),
		Services:      services,
	}
}
