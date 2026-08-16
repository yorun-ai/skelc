// Package schema defines skelc's normalized schema projection, persistence
// format, queries, and compatibility comparison.
package schema

import "go.yorun.ai/skelc/model"

const (
	Format        = "yorun.skel.schema"
	FormatVersion = 1
)

var declarationKinds = []string{
	"actor", "config", "data", "enum", "event", "resource", "service", "task", "web",
}

type Document struct {
	Format        string         `json:"format"`
	FormatVersion int            `json:"formatVersion"`
	Domain        string         `json:"domain"`
	Description   string         `json:"description,omitempty"`
	Declarations  []*Declaration `json:"declarations"`
}

type Metadata struct {
	Description      string `json:"description,omitempty"`
	Deprecated       bool   `json:"deprecated,omitempty"`
	DeprecatedReason string `json:"deprecatedReason,omitempty"`
}

type Declaration struct {
	Metadata
	Pub      bool            `json:"pub"`
	Name     string          `json:"name"`
	Kind     string          `json:"type"`
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

type EnumSchema struct {
	Items []*EnumItem `json:"items"`
}

type EnumItem struct {
	Metadata
	Name string         `json:"name"`
	Pos  model.Position `json:"-"`
}

type DataSchema struct {
	Lifecycle      string    `json:"lifecycle,omitempty"`
	Sensitive      bool      `json:"sensitive,omitempty"`
	TypeParameters []string  `json:"typeParameters,omitempty"`
	Members        []*Member `json:"members"`
}

type Member struct {
	Metadata
	Name      string         `json:"name"`
	Example   string         `json:"example,omitempty"`
	Sensitive bool           `json:"sensitive,omitempty"`
	Type      *Type          `json:"type"`
	Pos       model.Position `json:"-"`
}

type Type struct {
	Kind      string  `json:"kind"`
	Nullable  bool    `json:"nullable,omitempty"`
	Name      string  `json:"name,omitempty"`
	Arguments []*Type `json:"arguments,omitempty"`
	Element   *Type   `json:"element,omitempty"`
	Key       *Type   `json:"key,omitempty"`
	Value     *Type   `json:"value,omitempty"`
}

type ActorSchema struct {
	Vias           []*ActorVia `json:"vias"`
	AuthEnabled    bool        `json:"authEnabled,omitempty"`
	AuthCredential *DataSchema `json:"authCredential,omitempty"`
	AuthInfo       *DataSchema `json:"authInfo,omitempty"`
	PermEnabled    bool        `json:"permEnabled,omitempty"`
}

type ActorVia struct {
	Name string         `json:"name"`
	Pos  model.Position `json:"-"`
}

type ResourceSchema struct {
	Checks  []*ResourceCheck  `json:"checks,omitempty"`
	Actions []*ResourceAction `json:"actions"`
}

type ResourceAction struct {
	Metadata
	Name           string           `json:"name"`
	PermissionCode string           `json:"permissionCode"`
	Checks         []*ResourceCheck `json:"checks,omitempty"`
	Pos            model.Position   `json:"-"`
}

type ResourceCheck struct {
	Metadata
	Name      string         `json:"name"`
	Arguments []*Argument    `json:"arguments"`
	Pos       model.Position `json:"-"`
}

type ServiceSchema struct {
	Audiences []*Audience  `json:"audiences"`
	Auth      string       `json:"auth"`
	Require   *Requirement `json:"require,omitempty"`
	Methods   []*Method    `json:"methods"`
}

type Audience struct {
	Actor string         `json:"actor"`
	Via   string         `json:"via,omitempty"`
	Pos   model.Position `json:"-"`
}

type Method struct {
	Metadata
	Name               string         `json:"name"`
	SkelName           string         `json:"skelName"`
	Example            string         `json:"example,omitempty"`
	Auth               string         `json:"auth"`
	Require            *Requirement   `json:"require,omitempty"`
	InputDescription   string         `json:"inputDescription,omitempty"`
	ArgumentsSensitive bool           `json:"argumentsSensitive,omitempty"`
	OutputDescription  string         `json:"outputDescription,omitempty"`
	OutputExample      string         `json:"outputExample,omitempty"`
	ResultSensitive    bool           `json:"resultSensitive,omitempty"`
	Arguments          []*Argument    `json:"arguments"`
	Result             *Type          `json:"result,omitempty"`
	Pos                model.Position `json:"-"`
}

type Argument struct {
	Metadata
	Name      string         `json:"name"`
	Example   string         `json:"example,omitempty"`
	Sensitive bool           `json:"sensitive,omitempty"`
	Type      *Type          `json:"type"`
	Pos       model.Position `json:"-"`
}

type Requirement struct {
	Mode     string            `json:"mode"`
	Code     string            `json:"code,omitempty"`
	Check    *RequirementCheck `json:"check,omitempty"`
	Children []*Requirement    `json:"children,omitempty"`
}

type RequirementCheck struct {
	Resource  string                      `json:"resource"`
	Action    string                      `json:"action,omitempty"`
	Check     string                      `json:"check"`
	Arguments []*RequirementCheckArgument `json:"arguments,omitempty"`
}

type RequirementCheckArgument struct {
	Name     string `json:"name"`
	JSONPath string `json:"jsonPath"`
	Type     *Type  `json:"type"`
}

type WebSchema struct {
	Audiences []*Audience `json:"audiences"`
}

type TaskSchema struct {
	Triggers []*Trigger `json:"triggers"`
}

type Trigger struct {
	Metadata
	Name               string         `json:"name"`
	SkelName           string         `json:"skelName"`
	InputDescription   string         `json:"inputDescription,omitempty"`
	ArgumentsSensitive bool           `json:"argumentsSensitive,omitempty"`
	Arguments          []*Argument    `json:"arguments"`
	Pos                model.Position `json:"-"`
}

type Entry struct {
	Pub      bool   `json:"pub"`
	Name     string `json:"name"`
	Kind     string `json:"type"`
	SkelName string `json:"skelName"`
}
