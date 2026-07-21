package source

import (
	_ "embed"
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/internal/util/nameutil"
	"go.yorun.ai/skelc/model"
)

const serviceTsFilename = "service.ts"

//go:embed tpl/service.ts.tpl
var serviceTsTemplate string

type ServiceTsPayload struct {
	TypeImports         []string
	ExternalTypeImports []*TypeImport
	Services            []*Service
}

func (g *_Gen) genServiceTs() {
	payload := g.buildServiceTsPayload()
	g.renderTs(serviceTsFilename, serviceTsTemplate, payload)
}

func (g *_Gen) buildServiceTsPayload() *ServiceTsPayload {
	clientServices := g.serviceClientServices()
	typeImports := buildServiceTypeImports(clientServices)
	externalTypeImports := buildServiceExternalTypeImports(clientServices)
	payload := &ServiceTsPayload{
		Services:            make([]*Service, 0, len(clientServices)),
		ExternalTypeImports: externalTypeImports,
	}
	services := castServices(clientServices)
	payload.Services = services
	payload.TypeImports = typeImports
	return payload
}

type Service struct {
	Name         string
	SkelName     string
	SpecName     string
	CommentLines []string
	FactoryName  string
	Methods      []*ServiceMethod
	WireMethods  []*_WireMethod
}

type serviceNames struct {
	Name        string
	FactoryName string
	SpecName    string
}

func buildServiceNames(serviceName string) *serviceNames {
	name := nameutil.ToCamel(serviceName)
	return &serviceNames{
		Name:        name,
		FactoryName: fmt.Sprintf("create%s", name),
		SpecName:    fmt.Sprintf("%sSpec", name),
	}
}

func castService(p *model.Service) *Service {
	names := buildServiceNames(p.Name)
	service := &Service{
		Name:         names.Name,
		SkelName:     p.SkelName,
		SpecName:     names.SpecName,
		CommentLines: tsDocLines(p.Description),
		FactoryName:  names.FactoryName,
		Methods:      make([]*ServiceMethod, 0, len(p.Methods)),
	}
	for _, methodToken := range p.Methods {
		castedMethod := castServiceMethod(methodToken)
		service.Methods = append(service.Methods, castedMethod)
	}
	return service
}

func castServices(services []*model.Service) []*Service {
	castedServices := make([]*Service, 0, len(services))
	for _, serviceToken := range services {
		castedService := castService(serviceToken)
		castedServices = append(castedServices, castedService)
	}
	return castedServices
}

type ServiceMethod struct {
	Name         string
	SkelName     string
	SummaryLines []string
	ParamDocs    []*MethodParamDoc
	ReturnDoc    *MethodReturnDoc
	Arguments    []*MethodArgument
	HasParams    bool
	ResultType   *Type
	ReturnType   string
	HasWire      bool
}

func castServiceMethod(p *model.Method) *ServiceMethod {
	resultType := castType(p.ResultType)
	method := &ServiceMethod{
		Name:         nameutil.ToLowerCamel(p.Name),
		SkelName:     p.Name,
		SummaryLines: tsSummaryLines(p.Description, p.Example),
		ParamDocs:    make([]*MethodParamDoc, 0, len(p.Arguments)+2),
		Arguments:    make([]*MethodArgument, 0, len(p.Arguments)),
		HasParams:    len(p.Arguments) > 0,
		ResultType:   resultType,
		ReturnType:   "void",
		HasWire:      methodArgumentsContainBinary(p) || methodResultContainsBinary(p),
	}
	if resultType != nil {
		method.ReturnType = resultType.Plain
	}
	for _, argToken := range p.Arguments {
		castedArg := castMethodArgument(argToken)
		method.Arguments = append(method.Arguments, castedArg)
	}
	method.ParamDocs = append(method.ParamDocs, &MethodParamDoc{
		Name:        "params",
		Description: codegen.ChooseString(method.HasParams, "Request parameters", "Must be null"),
	})
	method.ParamDocs = append(method.ParamDocs, &MethodParamDoc{
		Name:        "options",
		Description: "Call options, optional",
	})
	method.ReturnDoc = tsReturnDoc(resultType, p.OutputDescription, p.OutputExample)
	return method
}

type MethodArgument struct {
	Name        string
	SkelName    string
	Description string
	Type        *Type
}

type MethodParamDoc struct {
	Name        string
	Description string
}

type MethodReturnDoc struct {
	TypeName    string
	Description string
}

func castMethodArgument(p *model.Argument) *MethodArgument {
	argType := castType(p.Type)
	return &MethodArgument{
		Name:        nameutil.ToLowerCamel(p.Name),
		SkelName:    p.Name,
		Description: tsTagDoc(codegen.MergeDescriptionAndExample(p.Description, p.Example)),
		Type:        argType,
	}
}

func buildServiceTypeImports(services []*model.Service) []string {
	imports := make([]string, 0)
	seen := make(map[string]struct{})
	for _, service := range services {
		for _, method := range service.Methods {
			imports = appendServiceTypeImports(imports, seen, method.ResultType)
			for _, argument := range method.Arguments {
				imports = appendServiceTypeImports(imports, seen, argument.Type)
			}
		}
	}
	return imports
}

func appendServiceTypeImports(imports []string, seen map[string]struct{}, typeToken *model.Type) []string {
	if typeToken == nil {
		return imports
	}

	switch typeToken.Kind {
	case model.TypeKindScalar, model.TypeKindTypeParameter:
		return imports
	case model.TypeKindList:
		return appendServiceTypeImports(imports, seen, typeToken.List.Value)
	case model.TypeKindMap:
		imports = appendServiceTypeImports(imports, seen, typeToken.Map.Key)
		return appendServiceTypeImports(imports, seen, typeToken.Map.Value)
	case model.TypeKindEnum:
		if typeToken.ExternalImportPath != "" {
			return imports
		}
		return appendUniqueServiceTypeImport(imports, seen, transEnumName(typeToken.Enum))
	case model.TypeKindData:
		if typeToken.ExternalImportPath == "" {
			imports = appendUniqueServiceTypeImport(imports, seen, transDataName(typeToken.Data))
		}
		for _, typeArgument := range typeToken.TypeArguments {
			imports = appendServiceTypeImports(imports, seen, typeArgument)
		}
		return imports
	}

	checkutil.Panicf("unexpected type %+v", typeToken)
	return nil
}

func appendUniqueServiceTypeImport(imports []string, seen map[string]struct{}, name string) []string {
	if _, ok := seen[name]; ok {
		return imports
	}
	seen[name] = struct{}{}
	return append(imports, name)
}

func buildServiceExternalTypeImports(services []*model.Service) []*TypeImport {
	imports := make([]*TypeImport, 0)
	seen := make(map[string]struct{})
	for _, service := range services {
		for _, method := range service.Methods {
			imports = appendExternalTypeImports(imports, seen, method.ResultType)
			for _, argument := range method.Arguments {
				imports = appendExternalTypeImports(imports, seen, argument.Type)
			}
		}
	}
	return imports
}

func tsDocLines(description string) []string {
	return codegen.SplitDocLines(description)
}

func tsCommentLines(description string, example string) []string {
	docLines := tsDocLines(codegen.MergeDescriptionAndExample(description, example))
	if len(docLines) == 0 {
		return nil
	}
	docLines[0] = codegen.EnsureSentence(docLines[0])
	return docLines
}

func tsSummaryLines(description string, example string) []string {
	return tsCommentLines(description, example)
}

func tsReturnDoc(resultType *Type, description string, example string) *MethodReturnDoc {
	if resultType == nil {
		return nil
	}
	return &MethodReturnDoc{
		TypeName:    resultType.Plain,
		Description: tsTagDoc(codegen.MergeDescriptionAndExample(description, example)),
	}
}

func tsTagDoc(description string) string {
	lines := tsDocLines(description)
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, " ")
}
