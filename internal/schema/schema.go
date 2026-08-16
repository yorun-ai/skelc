// Package schema projects semantic models into stable schema wire types and
// implements schema queries, persistence, validation, and compatibility diffs.
package schema

import publicschema "go.yorun.ai/skelc/schema"

const (
	Format        = publicschema.Format
	FormatVersion = publicschema.FormatVersion
)

var declarationKinds = []string{
	"actor", "config", "data", "enum", "event", "resource", "service", "task", "web",
}

type Document = publicschema.Document
type Metadata = publicschema.Metadata
type Declaration = publicschema.Declaration
type EnumSchema = publicschema.EnumSchema
type EnumItem = publicschema.EnumItem
type DataSchema = publicschema.DataSchema
type Member = publicschema.Member
type Type = publicschema.Type
type ActorSchema = publicschema.ActorSchema
type ActorVia = publicschema.ActorVia
type ResourceSchema = publicschema.ResourceSchema
type ResourceAction = publicschema.ResourceAction
type ResourceCheck = publicschema.ResourceCheck
type ServiceSchema = publicschema.ServiceSchema
type Audience = publicschema.Audience
type Method = publicschema.Method
type Argument = publicschema.Argument
type Requirement = publicschema.Requirement
type RequirementCheck = publicschema.RequirementCheck
type RequirementCheckArgument = publicschema.RequirementCheckArgument
type WebSchema = publicschema.WebSchema
type TaskSchema = publicschema.TaskSchema
type Trigger = publicschema.Trigger
type Entry = publicschema.Entry
