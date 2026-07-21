package model

// AuthMode controls whether authentication is required for a service or method.
type AuthMode string

const (
	// AuthModeUnset inherits authentication behavior from the enclosing context.
	AuthModeUnset AuthMode = "unset"
	// AuthModeAuth requires an authenticated actor.
	AuthModeAuth AuthMode = "auth"
	// AuthModeNoAuth explicitly allows unauthenticated access.
	AuthModeNoAuth AuthMode = "noauth"
)

// Service describes a callable service declaration.
type Service struct {
	// Pos is the service declaration's source position.
	Pos Position
	// Name is the service's local name.
	Name string
	// SkelName is the service's fully qualified Skel name.
	SkelName string
	// Hash is the service's compatibility hash.
	Hash string
	// Description is the service's documentation text.
	Description string
	// Pub reports whether the service belongs to the public contract.
	Pub bool
	// Audiences lists actors allowed to call the service.
	Audiences []*ActorAudience
	// Auth is the service-level authentication mode.
	Auth AuthMode
	// Require is the service-level permission requirement.
	Require *PermissionRequire
	// Methods lists methods in source order.
	Methods []*Method
}

// ActorAudience identifies an actor and transport allowed to access a service
// or web entry point.
type ActorAudience struct {
	// Actor is the referenced actor name as resolved from source.
	Actor string
	// Via is the required actor transport, or empty when no transport is selected.
	Via string
	// Pos is the audience declaration's source position.
	Pos Position
}

// Method describes one callable service method.
type Method struct {
	// Pos is the method declaration's source position.
	Pos Position
	// Name is the method's normalized local name.
	Name string
	// SkelName is the method name as represented in Skel metadata.
	SkelName string
	// Hash is the method's compatibility hash.
	Hash string
	// Description is the method's documentation text.
	Description string
	// Example is the method's example text.
	Example string
	// Auth is the method-level authentication mode.
	Auth AuthMode
	// Require is the method-level permission requirement.
	Require *PermissionRequire
	// Arguments lists input arguments in source order.
	Arguments []*Argument
	// ArgumentsData is the generated data model representing method arguments.
	ArgumentsData *Data
	// InputDescription documents the method input as a whole.
	InputDescription string
	// OutputDescription documents the method result.
	OutputDescription string
	// OutputExample is the method result's example text.
	OutputExample string
	// ResultType is the resolved result type, or nil for a method with no result.
	ResultType *Type
}

// Argument describes one service-method or task-trigger argument.
type Argument struct {
	// Pos is the argument's source position.
	Pos Position
	// Name is the argument's local name.
	Name string
	// Description is the argument's documentation text.
	Description string
	// Example is the argument's example value as source text.
	Example string
	// Type is the argument's resolved semantic type.
	Type *Type
}
