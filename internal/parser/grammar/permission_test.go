package grammar

import "testing"

func TestParseNestedPermissionRequire(t *testing.T) {
	content := parseSkelForTest(t, `
domain demo

service UserService {
    method updateProfile {
        require any(
            User:manage,
            all(
                User:read,
                User:update:byId(input.userId)
            )
        )
    }
}
`)

	method := serviceMethodsForTest(content.Entries[0].Service)[0]
	if method.Require == nil || method.Require.Expr.Mode != "any" {
		t.Fatalf("unexpected require expression: %+v", method.Require)
	}
	children := method.Require.Expr.Children
	if len(children) != 2 || children[1].Mode != "all" {
		t.Fatalf("unexpected require children: %+v", children)
	}
	check := children[1].Children[1].Term
	if check.Target.Resource.String() != "User" || check.Target.Action.Value != "update" {
		t.Fatalf("unexpected permission target: %+v", check.Target)
	}
	if check.Call == nil || check.Call.Name.Value != "byId" || check.Call.Arguments[0].String() != "input.userId" {
		t.Fatalf("unexpected permission call: %+v", check.Call)
	}
}
