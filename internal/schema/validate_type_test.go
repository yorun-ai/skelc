package schema

import (
	"strings"
	"testing"
)

// TODO: Remove this regression test after the PermissionCode migration period ends.
func TestValidateTypeRejectsRemovedPermissionCodeKind(t *testing.T) {
	err := validateType(&Type{Kind: TypeKind("permissionCode")})
	if err == nil || !strings.Contains(err.Error(), "unsupported type kind") {
		t.Fatalf("expected removed type kind to be rejected, got %v", err)
	}
}
