package schema

import (
	"bytes"
	"testing"
)

func TestWriteDeclarationTextData(t *testing.T) {
	declaration := &Declaration{
		Metadata: Metadata{Description: "User page", Deprecated: true, DeprecatedReason: "use AccountPage"},
		Pub:      true, Name: "UserPage", Kind: "data", SkelName: "demo.user.UserPage",
		Data: &DataSchema{
			Sensitive: true, TypeParameters: []string{"T"},
			Members: []*Member{{
				Metadata: Metadata{Description: "page value"}, Name: "value", Example: "one", Sensitive: true,
				Type: &Type{Kind: "reference", Name: "shared.Page", Nullable: true, Arguments: []*Type{{Kind: "typeParameter", Name: "T"}}},
			}},
		},
	}
	var output bytes.Buffer
	if err := WriteDeclarationText(&output, declaration); err != nil {
		t.Fatal(err)
	}
	expected := `pub data demo.user.UserPage
  name: UserPage
  description: "User page"
  deprecated: true
  deprecatedReason: "use AccountPage"
  sensitive: true
  typeParameters:
    - T
  members:
    - value: shared.Page<T>?
      description: "page value"
      example: "one"
      sensitive: true
`
	if output.String() != expected {
		t.Fatalf("unexpected declaration text:\n%s", output.String())
	}
}

func TestWriteDeclarationTextDeclarationKinds(t *testing.T) {
	tests := []struct {
		name        string
		declaration *Declaration
		expected    string
	}{
		{
			name: "enum",
			declaration: &Declaration{Name: "Status", Kind: "enum", SkelName: "demo.Status", Enum: &EnumSchema{
				Items: []*EnumItem{{Name: "ACTIVE"}},
			}},
			expected: "--- enum demo.Status\n  name: Status\n  items:\n    - ACTIVE\n",
		},
		{
			name: "actor",
			declaration: &Declaration{Pub: true, Name: "UserActor", Kind: "actor", SkelName: "demo.UserActor", Actor: &ActorSchema{
				Vias: []*ActorVia{{Name: "rpc"}},
			}},
			expected: "pub actor demo.UserActor\n  name: UserActor\n  vias:\n    - rpc\n",
		},
		{
			name: "resource",
			declaration: &Declaration{Pub: true, Name: "Order", Kind: "resource", SkelName: "demo.Order", Resource: &ResourceSchema{
				Actions: []*ResourceAction{{Name: "read", PermissionCode: "demo.Order:read"}},
			}},
			expected: "pub resource demo.Order\n  name: Order\n  actions:\n    - read\n      permissionCode: \"demo.Order:read\"\n",
		},
		{
			name: "service",
			declaration: &Declaration{Pub: true, Name: "UserService", Kind: "service", SkelName: "demo.UserService", Service: &ServiceSchema{
				Audiences: []*Audience{{Actor: "demo.UserActor", Via: "rpc"}}, Auth: "required",
				Require: &Requirement{Mode: "all", Children: []*Requirement{{Mode: "code", Code: "demo.User:read"}}},
				Methods: []*Method{{
					Name: "get", SkelName: "demo.UserService.get", Auth: "unset",
					Arguments: []*Argument{{Name: "id", Type: &Type{Kind: "scalar", Name: "uuid"}}},
					Result:    &Type{Kind: "data", Name: "demo.User"},
				}},
			}},
			expected: `pub service demo.UserService
  name: UserService
  audiences:
    - actor: demo.UserActor
      via: rpc
  auth: required
  require:
    mode: all
    children:
      - mode: code
        code: "demo.User:read"
  methods:
    - get
      skelName: demo.UserService.get
      auth: unset
      arguments:
        - id: uuid
      result: demo.User
`,
		},
		{
			name: "web",
			declaration: &Declaration{Name: "PortalWeb", Kind: "web", SkelName: "demo.PortalWeb", Web: &WebSchema{
				Audiences: []*Audience{},
			}},
			expected: "--- web demo.PortalWeb\n  name: PortalWeb\n  audiences: []\n",
		},
		{
			name: "task",
			declaration: &Declaration{Name: "CleanupTask", Kind: "task", SkelName: "demo.CleanupTask", Task: &TaskSchema{
				Triggers: []*Trigger{{Name: "run", SkelName: "demo.CleanupTask.run", Arguments: []*Argument{}}},
			}},
			expected: "--- task demo.CleanupTask\n  name: CleanupTask\n  triggers:\n    - run\n      skelName: demo.CleanupTask.run\n      arguments: []\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := WriteDeclarationText(&output, test.declaration); err != nil {
				t.Fatal(err)
			}
			if output.String() != test.expected {
				t.Fatalf("unexpected declaration text:\n%s", output.String())
			}
		})
	}
}
