// Package schema implements normalized schema projection, querying, encoding,
// validation, and compatibility diffing for the compiler and CLI.
package schema

import "go.yorun.ai/skelc/internal/model"

const (
	// Format identifies schema snapshot JSON produced by skelc.
	Format = "yorun.skel.schema"
	// FormatVersion is the current schema snapshot JSON format version.
	FormatVersion = 1
)

// ErrorCode identifies a schema command failure for programmatic consumers.
type ErrorCode string

const (
	ErrorCodeInvalidArgument    ErrorCode = "INVALID_ARGUMENT"
	ErrorCodeCompilationFailed  ErrorCode = "SCHEMA_COMPILATION_FAILED"
	ErrorCodeGitHistoryNotFound ErrorCode = "SCHEMA_GIT_HISTORY_NOT_FOUND"
	ErrorCodeCommandFailed      ErrorCode = "SCHEMA_COMMAND_FAILED"
)

// CommandError is emitted on stdout when a schema command fails.
type CommandError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// DeclarationType identifies a top-level Skel declaration in schema JSON.
type DeclarationType string

const (
	DeclarationTypeActor    DeclarationType = "actor"
	DeclarationTypeConfig   DeclarationType = "config"
	DeclarationTypeData     DeclarationType = "data"
	DeclarationTypeEnum     DeclarationType = "enum"
	DeclarationTypeEvent    DeclarationType = "event"
	DeclarationTypeResource DeclarationType = "resource"
	DeclarationTypeService  DeclarationType = "service"
	DeclarationTypeTask     DeclarationType = "task"
	DeclarationTypeWeb      DeclarationType = "web"
)

// ConfigLifecycle identifies the lifetime of a config declaration.
type ConfigLifecycle string

const (
	ConfigLifecycleEternal ConfigLifecycle = "eternal"
	ConfigLifecycleInstant ConfigLifecycle = "instant"
)

// TypeKind identifies the normalized representation carried by a Type.
type TypeKind string

const (
	TypeKindScalar            TypeKind = "scalar"
	TypeKindEnum              TypeKind = "enum"
	TypeKindData              TypeKind = "data"
	TypeKindConfig            TypeKind = "config"
	TypeKindEvent             TypeKind = "event"
	TypeKindTypeParameter     TypeKind = "typeParameter"
	TypeKindImportedReference TypeKind = "importedReference"
	TypeKindList              TypeKind = "list"
	TypeKindMap               TypeKind = "map"
	TypeKindPermissionCode    TypeKind = "permissionCode"
)

// AuthMode identifies the authentication behavior of a service or method.
type AuthMode string

const (
	AuthModeUnset  AuthMode = "unset"
	AuthModeAuth   AuthMode = "auth"
	AuthModeNoAuth AuthMode = "noauth"
)

// RequirementMode identifies one node in a normalized permission expression.
type RequirementMode string

const (
	RequirementModeCode      RequirementMode = "code"
	RequirementModeReference RequirementMode = "reference"
	RequirementModeCheck     RequirementMode = "check"
	RequirementModeAll       RequirementMode = "all"
	RequirementModeAny       RequirementMode = "any"
)

var declarationKinds = []DeclarationType{
	DeclarationTypeActor, DeclarationTypeConfig, DeclarationTypeData, DeclarationTypeEnum,
	DeclarationTypeEvent, DeclarationTypeResource, DeclarationTypeService, DeclarationTypeTask,
	DeclarationTypeWeb,
}

// Document is the versioned normalized domain emitted by schema snapshot.
type Document struct {
	Format        string         `json:"format"`
	FormatVersion int            `json:"formatVersion"`
	Domain        string         `json:"domain"`
	Description   string         `json:"description,omitempty"`
	Declarations  []*Declaration `json:"declarations"`
}

// Metadata contains common declaration and member documentation metadata.
type Metadata struct {
	Description      string `json:"description,omitempty"`
	Deprecated       bool   `json:"deprecated,omitempty"`
	DeprecatedReason string `json:"deprecatedReason,omitempty"`
}

// Declaration is one complete normalized declaration emitted by schema get or
// contained in a Document.
type Declaration struct {
	Metadata
	Pub      bool            `json:"pub"`
	Name     string          `json:"name"`
	Kind     DeclarationType `json:"type"`
	SkelName string          `json:"skelName"`
	Enum     *EnumSchema     `json:"enum,omitempty"`
	Data     *DataSchema     `json:"data,omitempty"`
	Actor    *ActorSchema    `json:"actor,omitempty"`
	Resource *ResourceSchema `json:"resource,omitempty"`
	Service  *ServiceSchema  `json:"service,omitempty"`
	Web      *WebSchema      `json:"web,omitempty"`
	Task     *TaskSchema     `json:"task,omitempty"`
	Pos      model.Position  `json:"-"`
}

// Entry is one declaration summary emitted by schema list.
type Entry struct {
	Pub      bool            `json:"pub"`
	Name     string          `json:"name"`
	Kind     DeclarationType `json:"type"`
	SkelName string          `json:"skelName"`
}
