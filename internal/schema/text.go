package schema

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// WriteDeclarationText writes a deterministic, human-readable schema detail
// tree for one declaration. JSON remains the lossless machine representation.
func WriteDeclarationText(writer io.Writer, declaration *Declaration) error {
	if declaration == nil {
		return fmt.Errorf("schema declaration is required")
	}
	if declaration.Name == "" || declaration.SkelName == "" || declaration.Kind == "" {
		return fmt.Errorf("schema declaration has an empty name or type")
	}
	if err := validateDeclarationTextBody(declaration); err != nil {
		return fmt.Errorf("schema declaration %s: %w", declaration.SkelName, err)
	}

	visibility := "---"
	if declaration.Pub {
		visibility = "pub"
	}
	textWriter := &_DeclarationTextWriter{}
	textWriter.line(0, "%s %s %s", visibility, declaration.Kind, declaration.SkelName)
	textWriter.line(1, "name: %s", declaration.Name)
	textWriter.metadata(1, declaration.Metadata)
	switch declaration.Kind {
	case "enum":
		textWriter.enum(declaration.Enum, 1)
	case "data", "config", "event":
		textWriter.data(declaration.Data, 1)
	case "actor":
		textWriter.actor(declaration.Actor, 1)
	case "resource":
		textWriter.resource(declaration.Resource, 1)
	case "service":
		textWriter.service(declaration.Service, 1)
	case "web":
		textWriter.audiences(declaration.Web.Audiences, 1)
	case "task":
		textWriter.task(declaration.Task, 1)
	}
	if _, err := io.WriteString(writer, textWriter.builder.String()); err != nil {
		return fmt.Errorf("write schema declaration: %w", err)
	}
	return nil
}

func validateDeclarationTextBody(declaration *Declaration) error {
	valid := false
	switch declaration.Kind {
	case "enum":
		valid = declaration.Enum != nil
	case "data", "config", "event":
		valid = declaration.Data != nil
	case "actor":
		valid = declaration.Actor != nil
	case "resource":
		valid = declaration.Resource != nil
	case "service":
		valid = declaration.Service != nil
	case "web":
		valid = declaration.Web != nil
	case "task":
		valid = declaration.Task != nil
	}
	if !valid {
		return fmt.Errorf("declaration body does not match type %q", declaration.Kind)
	}
	return nil
}

type _DeclarationTextWriter struct {
	builder strings.Builder
}

func (w *_DeclarationTextWriter) line(indent int, format string, args ...any) {
	w.builder.WriteString(strings.Repeat("  ", indent))
	_, _ = fmt.Fprintf(&w.builder, format, args...)
	w.builder.WriteByte('\n')
}

func (w *_DeclarationTextWriter) metadata(indent int, value Metadata) {
	if value.Description != "" {
		w.line(indent, "description: %s", strconv.Quote(value.Description))
	}
	if value.Deprecated {
		w.line(indent, "deprecated: true")
	}
	if value.DeprecatedReason != "" {
		w.line(indent, "deprecatedReason: %s", strconv.Quote(value.DeprecatedReason))
	}
}

func (w *_DeclarationTextWriter) enum(value *EnumSchema, indent int) {
	if len(value.Items) == 0 {
		w.line(indent, "items: []")
		return
	}
	w.line(indent, "items:")
	for _, item := range value.Items {
		w.line(indent+1, "- %s", item.Name)
		w.metadata(indent+2, item.Metadata)
	}
}

func (w *_DeclarationTextWriter) data(value *DataSchema, indent int) {
	if value.Lifecycle != "" {
		w.line(indent, "lifecycle: %s", value.Lifecycle)
	}
	if value.Sensitive {
		w.line(indent, "sensitive: true")
	}
	if len(value.TypeParameters) > 0 {
		w.line(indent, "typeParameters:")
		for _, parameter := range value.TypeParameters {
			w.line(indent+1, "- %s", parameter)
		}
	}
	w.members(value.Members, indent)
}

func (w *_DeclarationTextWriter) members(values []*Member, indent int) {
	if len(values) == 0 {
		w.line(indent, "members: []")
		return
	}
	w.line(indent, "members:")
	for _, member := range values {
		w.line(indent+1, "- %s: %s", member.Name, typeDisplay(member.Type))
		w.metadata(indent+2, member.Metadata)
		if member.Example != "" {
			w.line(indent+2, "example: %s", strconv.Quote(member.Example))
		}
		if member.Sensitive {
			w.line(indent+2, "sensitive: true")
		}
	}
}

func (w *_DeclarationTextWriter) actor(value *ActorSchema, indent int) {
	if len(value.Vias) == 0 {
		w.line(indent, "vias: []")
	} else {
		w.line(indent, "vias:")
		for _, via := range value.Vias {
			w.line(indent+1, "- %s", via.Name)
		}
	}
	if value.AuthEnabled {
		w.line(indent, "authEnabled: true")
	}
	if value.AuthCredential != nil {
		w.line(indent, "authCredential:")
		w.data(value.AuthCredential, indent+1)
	}
	if value.AuthInfo != nil {
		w.line(indent, "authInfo:")
		w.data(value.AuthInfo, indent+1)
	}
	if value.PermEnabled {
		w.line(indent, "permEnabled: true")
	}
}

func (w *_DeclarationTextWriter) resource(value *ResourceSchema, indent int) {
	if len(value.Checks) > 0 {
		w.resourceChecks("checks", value.Checks, indent)
	}
	if len(value.Actions) == 0 {
		w.line(indent, "actions: []")
		return
	}
	w.line(indent, "actions:")
	for _, action := range value.Actions {
		w.line(indent+1, "- %s", action.Name)
		w.metadata(indent+2, action.Metadata)
		w.line(indent+2, "permissionCode: %s", strconv.Quote(action.PermissionCode))
		if len(action.Checks) > 0 {
			w.resourceChecks("checks", action.Checks, indent+2)
		}
	}
}

func (w *_DeclarationTextWriter) resourceChecks(label string, values []*ResourceCheck, indent int) {
	w.line(indent, "%s:", label)
	for _, check := range values {
		w.line(indent+1, "- %s", check.Name)
		w.metadata(indent+2, check.Metadata)
		w.arguments("arguments", check.Arguments, indent+2)
	}
}

func (w *_DeclarationTextWriter) service(value *ServiceSchema, indent int) {
	w.audiences(value.Audiences, indent)
	w.line(indent, "auth: %s", value.Auth)
	if value.Require != nil {
		w.line(indent, "require:")
		w.requirement(value.Require, indent+1, false)
	}
	if len(value.Methods) == 0 {
		w.line(indent, "methods: []")
		return
	}
	w.line(indent, "methods:")
	for _, method := range value.Methods {
		w.line(indent+1, "- %s", method.Name)
		w.line(indent+2, "skelName: %s", method.SkelName)
		w.metadata(indent+2, method.Metadata)
		if method.Example != "" {
			w.line(indent+2, "example: %s", strconv.Quote(method.Example))
		}
		w.line(indent+2, "auth: %s", method.Auth)
		if method.Require != nil {
			w.line(indent+2, "require:")
			w.requirement(method.Require, indent+3, false)
		}
		if method.InputDescription != "" {
			w.line(indent+2, "inputDescription: %s", strconv.Quote(method.InputDescription))
		}
		if method.ArgumentsSensitive {
			w.line(indent+2, "argumentsSensitive: true")
		}
		if method.OutputDescription != "" {
			w.line(indent+2, "outputDescription: %s", strconv.Quote(method.OutputDescription))
		}
		if method.OutputExample != "" {
			w.line(indent+2, "outputExample: %s", strconv.Quote(method.OutputExample))
		}
		if method.ResultSensitive {
			w.line(indent+2, "resultSensitive: true")
		}
		w.arguments("arguments", method.Arguments, indent+2)
		w.line(indent+2, "result: %s", typeDisplay(method.Result))
	}
}

func (w *_DeclarationTextWriter) audiences(values []*Audience, indent int) {
	if len(values) == 0 {
		w.line(indent, "audiences: []")
		return
	}
	w.line(indent, "audiences:")
	for _, audience := range values {
		w.line(indent+1, "- actor: %s", audience.Actor)
		if audience.Via != "" {
			w.line(indent+2, "via: %s", audience.Via)
		}
	}
}

func (w *_DeclarationTextWriter) arguments(label string, values []*Argument, indent int) {
	if len(values) == 0 {
		w.line(indent, "%s: []", label)
		return
	}
	w.line(indent, "%s:", label)
	for _, argument := range values {
		w.line(indent+1, "- %s: %s", argument.Name, typeDisplay(argument.Type))
		w.metadata(indent+2, argument.Metadata)
		if argument.Example != "" {
			w.line(indent+2, "example: %s", strconv.Quote(argument.Example))
		}
		if argument.Sensitive {
			w.line(indent+2, "sensitive: true")
		}
	}
}

func (w *_DeclarationTextWriter) requirement(value *Requirement, indent int, listItem bool) {
	mode := value.Mode
	if mode == "" && value.Check != nil {
		mode = "reference"
	}
	prefix := "mode:"
	if listItem {
		prefix = "- mode:"
	}
	w.line(indent, "%s %s", prefix, mode)
	detailIndent := indent
	if listItem {
		detailIndent++
	}
	if value.Code != "" {
		w.line(detailIndent, "code: %s", strconv.Quote(value.Code))
	}
	if value.Check != nil {
		w.line(detailIndent, "check:")
		w.line(detailIndent+1, "resource: %s", value.Check.Resource)
		if value.Check.Action != "" {
			w.line(detailIndent+1, "action: %s", value.Check.Action)
		}
		if value.Check.Check != "" {
			w.line(detailIndent+1, "check: %s", value.Check.Check)
		}
		if len(value.Check.Arguments) > 0 {
			w.line(detailIndent+1, "arguments:")
			for _, argument := range value.Check.Arguments {
				if argument.Name != "" {
					w.line(detailIndent+2, "- %s: %s", argument.Name, typeDisplay(argument.Type))
					if argument.JSONPath != "" {
						w.line(detailIndent+3, "jsonPath: %s", strconv.Quote(argument.JSONPath))
					}
					continue
				}
				w.line(detailIndent+2, "- jsonPath: %s", strconv.Quote(argument.JSONPath))
				if argument.Type != nil {
					w.line(detailIndent+3, "type: %s", typeDisplay(argument.Type))
				}
			}
		}
	}
	if len(value.Children) > 0 {
		w.line(detailIndent, "children:")
		for _, child := range value.Children {
			w.requirement(child, detailIndent+1, true)
		}
	}
}

func (w *_DeclarationTextWriter) task(value *TaskSchema, indent int) {
	if len(value.Triggers) == 0 {
		w.line(indent, "triggers: []")
		return
	}
	w.line(indent, "triggers:")
	for _, trigger := range value.Triggers {
		w.line(indent+1, "- %s", trigger.Name)
		w.line(indent+2, "skelName: %s", trigger.SkelName)
		w.metadata(indent+2, trigger.Metadata)
		if trigger.InputDescription != "" {
			w.line(indent+2, "inputDescription: %s", strconv.Quote(trigger.InputDescription))
		}
		if trigger.ArgumentsSensitive {
			w.line(indent+2, "argumentsSensitive: true")
		}
		w.arguments("arguments", trigger.Arguments, indent+2)
	}
}
