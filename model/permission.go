package model

// PermissionRequireMode identifies the form of a permission expression.
type PermissionRequireMode string

const (
	// PermissionRequireModeCode requires one resource action code.
	PermissionRequireModeCode PermissionRequireMode = "code"
	// PermissionRequireModeCheck invokes a resource permission check.
	PermissionRequireModeCheck PermissionRequireMode = "check"
	// PermissionRequireModeAll requires every child expression to pass.
	PermissionRequireModeAll PermissionRequireMode = "all"
	// PermissionRequireModeAny requires at least one child expression to pass.
	PermissionRequireModeAny PermissionRequireMode = "any"
)

// PermissionRequire describes a normalized require clause.
type PermissionRequire struct {
	// Expr is the root of the permission expression tree.
	Expr *PermissionExpr
}

// PermissionExpr is one node in a normalized permission expression tree.
type PermissionExpr struct {
	// Mode identifies which of Code, Check, or Children carries this node's value.
	Mode PermissionRequireMode
	// Code is the fully qualified resource action code for code expressions.
	Code string
	// Check describes the invocation for check expressions.
	Check *PermissionCheckInvocation
	// Children contains operands for all and any expressions.
	Children []*PermissionExpr
}

// PermissionCheckInvocation describes a resolved resource permission-check
// method call.
type PermissionCheckInvocation struct {
	// ResourceSkelName is the resource's fully qualified Skel name.
	ResourceSkelName string
	// ActionName is the resource action's local name.
	ActionName string
	// CheckName is the resource check's local name.
	CheckName string
	// ServiceSkelName is the generated check service's fully qualified Skel name.
	ServiceSkelName string
	// MethodSkelName is the generated check method's Skel name.
	MethodSkelName string
	// Arguments lists resolved invocation arguments in method order.
	Arguments []*PermissionCheckArgument
}

// PermissionCheckArgument describes one resolved permission-check argument.
type PermissionCheckArgument struct {
	// Name is the target method argument name.
	Name string
	// JsonPath selects the value supplied to the argument.
	JsonPath string
	// Type is the argument's resolved semantic type.
	Type *Type
}
