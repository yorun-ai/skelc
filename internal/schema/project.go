package schema

import (
	"fmt"
	"slices"
	"strings"

	"go.yorun.ai/skelc/model"
)

const unresolvedReferenceTypeKind model.TypeKind = -1

func Project(domain *model.Domain, importAliases map[string]string) (*Document, error) {
	if domain == nil {
		return nil, fmt.Errorf("cannot project a nil domain")
	}
	document := &Document{
		Format: Format, FormatVersion: FormatVersion, Domain: domain.Name(),
		Description: domain.Description(), Declarations: []*Declaration{},
	}
	aliases := make(map[string]string, len(importAliases)+len(domain.Imports()))
	for alias, name := range importAliases {
		aliases[alias] = name
	}
	for _, imported := range domain.Imports() {
		aliases[imported.Alias] = imported.Name
	}
	appendDeclarations(document, domain, aliases, domain.Enums(), domain.Data(), domain.Configs(), domain.Events(),
		domain.Actors(), domain.Resources(), domain.Services(), domain.Webs(), domain.Tasks())
	normalizeReferenceNames(document, domain.Name(), aliases)
	slices.SortFunc(document.Declarations, compareDeclarations)
	return document, nil
}

func ValidateKind(kind string) error {
	if slices.Contains(declarationKinds, kind) {
		return nil
	}
	return fmt.Errorf("invalid schema declaration type %q, expected %s", kind, strings.Join(declarationKinds, "/"))
}

func appendDeclarations(
	document *Document,
	domain *model.Domain,
	importAliases map[string]string,
	enums []*model.Enum,
	data []*model.Data,
	configs []*model.Data,
	events []*model.Data,
	actors []*model.Actor,
	resources []*model.Resource,
	services []*model.Service,
	webs []*model.Web,
	tasks []*model.Task,
) {
	for _, value := range enums {
		document.Declarations = append(document.Declarations, projectEnum(value))
	}
	for _, value := range data {
		document.Declarations = append(document.Declarations, projectData(value))
	}
	for _, value := range configs {
		document.Declarations = append(document.Declarations, projectData(value))
	}
	for _, value := range events {
		document.Declarations = append(document.Declarations, projectData(value))
	}
	for _, value := range actors {
		document.Declarations = append(document.Declarations, projectActor(value))
	}
	for _, value := range resources {
		document.Declarations = append(document.Declarations, projectResource(value))
	}
	for _, value := range services {
		document.Declarations = append(document.Declarations, projectService(domain.Name(), importAliases, value))
	}
	for _, value := range webs {
		document.Declarations = append(document.Declarations, projectWeb(domain.Name(), importAliases, value))
	}
	for _, value := range tasks {
		document.Declarations = append(document.Declarations, projectTask(value))
	}
}

func projectEnum(value *model.Enum) *Declaration {
	items := make([]*EnumItem, 0, len(value.Items))
	for _, item := range value.Items {
		items = append(items, &EnumItem{Metadata: metadata(item.Description, item.Deprecated, item.DeprecatedReason), Name: item.Name, Pos: item.Pos})
	}
	return &Declaration{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Pub:      value.Pub, Name: value.Name, Kind: "enum", SkelName: value.SkelName, Pos: value.Pos,
		Enum: &EnumSchema{Items: items},
	}
}

func projectData(value *model.Data) *Declaration {
	return &Declaration{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Pub:      value.Pub, Name: value.Name, Kind: string(value.Kind), SkelName: value.SkelName, Pos: value.Pos,
		Data: projectDataSchema(value),
	}
}

func projectDataSchema(value *model.Data) *DataSchema {
	if value == nil {
		return nil
	}
	typeParameters := make([]string, 0, len(value.TypeParameters))
	for _, parameter := range value.TypeParameters {
		typeParameters = append(typeParameters, parameter.Name)
	}
	return &DataSchema{
		Lifecycle: string(value.Lifecycle), Sensitive: value.Sensitive,
		TypeParameters: typeParameters, Members: projectMembers(value.Members),
	}
}

func projectMembers(values []*model.DataMember) []*Member {
	members := make([]*Member, 0, len(values))
	for _, value := range values {
		members = append(members, &Member{
			Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
			Name:     value.Name, Example: value.Example, Sensitive: value.Sensitive, Type: projectType(value.Type), Pos: value.Pos,
		})
	}
	return members
}

func projectActor(value *model.Actor) *Declaration {
	vias := make([]*ActorVia, 0, len(value.Vias))
	for _, via := range value.Vias {
		vias = append(vias, &ActorVia{Name: via.Name, Pos: via.Pos})
	}
	return &Declaration{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Pub:      value.Pub, Name: value.Name, Kind: "actor", SkelName: value.SkelName, Pos: value.Pos,
		Actor: &ActorSchema{
			Vias: vias, AuthEnabled: value.AuthEnabled, AuthCredential: projectDataSchema(value.AuthCredential),
			AuthInfo: projectDataSchema(value.AuthInfo), PermEnabled: value.PermEnabled,
		},
	}
}

func projectResource(value *model.Resource) *Declaration {
	actions := make([]*ResourceAction, 0, len(value.Actions))
	for _, action := range value.Actions {
		actions = append(actions, &ResourceAction{
			Metadata: metadata(action.Description, action.Deprecated, action.DeprecatedReason),
			Name:     action.Name, PermissionCode: action.PermissionCode, Checks: projectResourceChecks(action.Checks), Pos: action.Pos,
		})
	}
	return &Declaration{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Pub:      value.Pub, Name: value.Name, Kind: "resource", SkelName: value.SkelName, Pos: value.Pos,
		Resource: &ResourceSchema{Checks: projectResourceChecks(value.Checks), Actions: actions},
	}
}

func projectResourceChecks(values []*model.ResourceCheck) []*ResourceCheck {
	checks := make([]*ResourceCheck, 0, len(values))
	for _, value := range values {
		checks = append(checks, &ResourceCheck{
			Metadata: metadata(value.Method.Description, value.Deprecated, value.DeprecatedReason),
			Name:     value.Name, Arguments: projectArguments(value.Method.Arguments), Pos: value.Method.Pos,
		})
	}
	return checks
}

func projectService(domainName string, importAliases map[string]string, value *model.Service) *Declaration {
	methods := make([]*Method, 0, len(value.Methods))
	for _, method := range value.Methods {
		methods = append(methods, projectMethod(method))
	}
	return &Declaration{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Pub:      value.Pub, Name: value.Name, Kind: "service", SkelName: value.SkelName, Pos: value.Pos,
		Service: &ServiceSchema{
			Audiences: projectAudiences(domainName, importAliases, value.Audiences), Auth: normalizedAuth(value.Auth),
			Require: projectRequirement(value.Require), Methods: methods,
		},
	}
}

func projectMethod(value *model.Method) *Method {
	return &Method{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Name:     value.Name, SkelName: value.SkelName, Example: value.Example, Auth: normalizedAuth(value.Auth),
		Require: projectRequirement(value.Require), InputDescription: value.InputDescription,
		ArgumentsSensitive: value.ArgumentsSensitive, OutputDescription: value.OutputDescription,
		OutputExample: value.OutputExample, ResultSensitive: value.ResultSensitive,
		Arguments: projectArguments(value.Arguments), Result: projectType(value.ResultType), Pos: value.Pos,
	}
}

func projectArguments(values []*model.Argument) []*Argument {
	arguments := make([]*Argument, 0, len(values))
	for _, value := range values {
		arguments = append(arguments, &Argument{
			Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
			Name:     value.Name, Example: value.Example, Sensitive: value.Sensitive, Type: projectType(value.Type), Pos: value.Pos,
		})
	}
	return arguments
}

func projectRequirement(value *model.PermissionRequire) *Requirement {
	if value == nil {
		return nil
	}
	return projectRequirementExpr(value.Expr)
}

func projectRequirementExpr(value *model.PermissionExpr) *Requirement {
	if value == nil {
		return nil
	}
	mode := string(value.Mode)
	if mode == "" && value.Check != nil {
		mode = "reference"
	}
	result := &Requirement{Mode: mode, Code: value.Code}
	if value.Check != nil {
		arguments := make([]*RequirementCheckArgument, 0, len(value.Check.Arguments))
		for _, argument := range value.Check.Arguments {
			arguments = append(arguments, &RequirementCheckArgument{Name: argument.Name, JSONPath: argument.JsonPath, Type: projectType(argument.Type)})
		}
		result.Check = &RequirementCheck{
			Resource: value.Check.ResourceSkelName, Action: value.Check.ActionName,
			Check: value.Check.CheckName, Arguments: arguments,
		}
	}
	for _, child := range value.Children {
		result.Children = append(result.Children, projectRequirementExpr(child))
	}
	return result
}

func projectWeb(domainName string, importAliases map[string]string, value *model.Web) *Declaration {
	return &Declaration{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Name:     value.Name, Kind: "web", SkelName: value.SkelName, Pos: value.Pos,
		Web: &WebSchema{Audiences: projectAudiences(domainName, importAliases, value.Audiences)},
	}
}

func projectTask(value *model.Task) *Declaration {
	triggers := make([]*Trigger, 0, len(value.Triggers))
	for _, trigger := range value.Triggers {
		triggers = append(triggers, &Trigger{
			Metadata: metadata(trigger.Description, trigger.Deprecated, trigger.DeprecatedReason),
			Name:     trigger.Name, SkelName: trigger.SkelName, InputDescription: trigger.InputDescription,
			ArgumentsSensitive: trigger.ArgumentsSensitive, Arguments: projectArguments(trigger.Arguments), Pos: trigger.Pos,
		})
	}
	return &Declaration{
		Metadata: metadata(value.Description, value.Deprecated, value.DeprecatedReason),
		Name:     value.Name, Kind: "task", SkelName: value.SkelName, Pos: value.Pos,
		Task: &TaskSchema{Triggers: triggers},
	}
}

func projectAudiences(domainName string, importAliases map[string]string, values []*model.ActorAudience) []*Audience {
	audiences := make([]*Audience, 0, len(values))
	for _, value := range values {
		audiences = append(audiences, &Audience{Actor: canonicalReferenceName(domainName, importAliases, value.Actor), Via: value.Via, Pos: value.Pos})
	}
	return audiences
}

func projectType(value *model.Type) *Type {
	if value == nil {
		return nil
	}
	result := &Type{Nullable: value.Nullable}
	switch value.Kind {
	case unresolvedReferenceTypeKind:
		result.Kind = "importedReference"
		result.Name = value.SkelName
		if value.ExternalAlias != "" {
			result.Name = value.ExternalAlias + "." + value.SkelName
		}
	case model.TypeKindScalar:
		result.Kind = "scalar"
		result.Name = strings.ToLower(value.Scalar.Name())
	case model.TypeKindList:
		result.Kind = "list"
		result.Element = projectType(value.List.Value)
	case model.TypeKindMap:
		result.Kind = "map"
		result.Key = projectType(value.Map.Key)
		result.Value = projectType(value.Map.Value)
	case model.TypeKindEnum:
		result.Kind = "enum"
		result.Name = value.SkelName
	case model.TypeKindData:
		result.Kind = "data"
		result.Name = value.SkelName
	case model.TypeKindTypeParameter:
		result.Kind = "typeParameter"
		if value.TypeParameter != nil {
			result.Name = value.TypeParameter.Name
		}
	case model.TypeKindSkelPermissionCode:
		result.Kind = "permissionCode"
	default:
		result.Kind = fmt.Sprintf("unknown:%d", value.Kind)
	}
	for _, argument := range value.TypeArguments {
		result.Arguments = append(result.Arguments, projectType(argument))
	}
	return result
}

func metadata(description string, deprecated bool, reason string) Metadata {
	return Metadata{Description: description, Deprecated: deprecated, DeprecatedReason: reason}
}

func normalizedAuth(value model.AuthMode) string {
	if value == "" {
		return string(model.AuthModeUnset)
	}
	return string(value)
}

func normalizeReferenceNames(document *Document, domainName string, importAliases map[string]string) {
	for _, declaration := range document.Declarations {
		normalizeDeclarationReferences(declaration, domainName, importAliases)
	}
}

func normalizeDeclarationReferences(declaration *Declaration, domainName string, importAliases map[string]string) {
	if declaration.Data != nil {
		normalizeDataReferences(declaration.Data, domainName, importAliases)
	}
	if declaration.Actor != nil {
		normalizeDataReferences(declaration.Actor.AuthCredential, domainName, importAliases)
		normalizeDataReferences(declaration.Actor.AuthInfo, domainName, importAliases)
	}
	if declaration.Resource != nil {
		normalizeResourceCheckReferences(declaration.Resource.Checks, domainName, importAliases)
		for _, action := range declaration.Resource.Actions {
			normalizeResourceCheckReferences(action.Checks, domainName, importAliases)
		}
	}
	if declaration.Service != nil {
		normalizeRequirementReferences(declaration.Service.Require, domainName, importAliases)
		for _, method := range declaration.Service.Methods {
			normalizeArgumentReferences(method.Arguments, domainName, importAliases)
			normalizeTypeReference(method.Result, domainName, importAliases)
			normalizeRequirementReferences(method.Require, domainName, importAliases)
		}
	}
	if declaration.Task != nil {
		for _, trigger := range declaration.Task.Triggers {
			normalizeArgumentReferences(trigger.Arguments, domainName, importAliases)
		}
	}
}

func normalizeDataReferences(value *DataSchema, domainName string, importAliases map[string]string) {
	if value == nil {
		return
	}
	for _, member := range value.Members {
		normalizeTypeReference(member.Type, domainName, importAliases)
	}
}

func normalizeResourceCheckReferences(values []*ResourceCheck, domainName string, importAliases map[string]string) {
	for _, check := range values {
		normalizeArgumentReferences(check.Arguments, domainName, importAliases)
	}
}

func normalizeArgumentReferences(values []*Argument, domainName string, importAliases map[string]string) {
	for _, argument := range values {
		normalizeTypeReference(argument.Type, domainName, importAliases)
	}
}

func normalizeTypeReference(value *Type, domainName string, importAliases map[string]string) {
	if value == nil {
		return
	}
	if value.Kind == "importedReference" {
		value.Name = canonicalReferenceName(domainName, importAliases, value.Name)
	}
	for _, argument := range value.Arguments {
		normalizeTypeReference(argument, domainName, importAliases)
	}
	normalizeTypeReference(value.Element, domainName, importAliases)
	normalizeTypeReference(value.Key, domainName, importAliases)
	normalizeTypeReference(value.Value, domainName, importAliases)
}

func normalizeRequirementReferences(value *Requirement, domainName string, importAliases map[string]string) {
	if value == nil {
		return
	}
	if value.Check != nil {
		value.Check.Resource = canonicalReferenceName(domainName, importAliases, value.Check.Resource)
		for _, argument := range value.Check.Arguments {
			normalizeTypeReference(argument.Type, domainName, importAliases)
		}
	}
	for _, child := range value.Children {
		normalizeRequirementReferences(child, domainName, importAliases)
	}
}

func canonicalReferenceName(domainName string, importAliases map[string]string, name string) string {
	if name == "" {
		return ""
	}
	if alias, localName, ok := strings.Cut(name, "."); ok {
		if importedDomain := importAliases[alias]; importedDomain != "" {
			return importedDomain + "." + localName
		}
		return name
	}
	return domainName + "." + name
}

func compareDeclarations(left, right *Declaration) int {
	leftOrder := kindOrder(left.Kind)
	rightOrder := kindOrder(right.Kind)
	if leftOrder != rightOrder {
		return leftOrder - rightOrder
	}
	return strings.Compare(left.SkelName, right.SkelName)
}

func kindOrder(kind string) int {
	switch kind {
	case "actor":
		return 1
	case "config":
		return 2
	case "data":
		return 3
	case "enum":
		return 4
	case "event":
		return 5
	case "resource":
		return 6
	case "service":
		return 7
	case "task":
		return 8
	case "web":
		return 9
	default:
		return 99
	}
}
