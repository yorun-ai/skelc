package schema

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"go.yorun.ai/skelc/model"
)

type Impact string

const (
	ImpactBreaking   Impact = "breaking"
	ImpactDangerous  Impact = "dangerous"
	ImpactCompatible Impact = "compatible"
)

type Change struct {
	Code      string          `json:"code"`
	Impact    Impact          `json:"impact"`
	Symbol    string          `json:"symbol"`
	Message   string          `json:"message"`
	Baseline  *model.Position `json:"baseline,omitempty"`
	Candidate *model.Position `json:"candidate,omitempty"`
}

type Summary struct {
	Breaking   int `json:"breaking"`
	Dangerous  int `json:"dangerous"`
	Compatible int `json:"compatible"`
}

type Report struct {
	Compatible      bool      `json:"compatible"`
	BaselineDomain  string    `json:"baselineDomain"`
	CandidateDomain string    `json:"candidateDomain"`
	Scope           Scope     `json:"scope"`
	Summary         Summary   `json:"summary"`
	Changes         []*Change `json:"changes"`
}

func Compare(baseline, candidate *Document) (*Report, error) {
	if err := Validate(baseline); err != nil {
		return nil, fmt.Errorf("baseline: %w", err)
	}
	if err := Validate(candidate); err != nil {
		return nil, fmt.Errorf("candidate: %w", err)
	}
	if baseline.Scope != candidate.Scope {
		return nil, fmt.Errorf("schema scope mismatch: baseline is %s, candidate is %s", baseline.Scope, candidate.Scope)
	}
	comparison := &_Comparison{report: &Report{
		Compatible: true, BaselineDomain: baseline.Domain, CandidateDomain: candidate.Domain,
		Scope: candidate.Scope, Changes: []*Change{},
	}}
	if baseline.Domain != candidate.Domain {
		comparison.add(ImpactBreaking, "domain.name.changed", candidate.Domain,
			fmt.Sprintf("domain name changed from %s to %s", baseline.Domain, candidate.Domain), model.Position{}, model.Position{})
		comparison.finish()
		return comparison.report, nil
	}

	candidateByName := declarationsByKey(candidate.Declarations)
	matchedBaseline := map[*Declaration]bool{}
	matchedCandidate := map[*Declaration]bool{}
	for _, declaration := range baseline.Declarations {
		other := candidateByName[declarationKey(declaration)]
		if other == nil {
			continue
		}
		comparison.compareDeclaration(declaration, other)
		matchedBaseline[declaration] = true
		matchedCandidate[other] = true
	}
	baselineUnmatched := unmatchedDeclarationsByName(baseline.Declarations, matchedBaseline)
	candidateUnmatched := unmatchedDeclarationsByName(candidate.Declarations, matchedCandidate)
	for skelName, baselineValues := range baselineUnmatched {
		candidateValues := candidateUnmatched[skelName]
		if len(baselineValues) != 1 || len(candidateValues) != 1 {
			continue
		}
		comparison.compareDeclaration(baselineValues[0], candidateValues[0])
		matchedBaseline[baselineValues[0]] = true
		matchedCandidate[candidateValues[0]] = true
	}
	for _, declaration := range baseline.Declarations {
		if matchedBaseline[declaration] {
			continue
		}
		comparison.add(ImpactBreaking, "declaration.removed", declaration.SkelName,
			fmt.Sprintf("%s %s was removed", declaration.Kind, declaration.SkelName), declaration.Pos, model.Position{})
	}
	for _, declaration := range candidate.Declarations {
		if matchedCandidate[declaration] {
			continue
		}
		comparison.add(ImpactCompatible, "declaration.added", declaration.SkelName,
			fmt.Sprintf("%s %s was added", declaration.Kind, declaration.SkelName), model.Position{}, declaration.Pos)
	}
	comparison.finish()
	return comparison.report, nil
}

func (r *Report) HasImpact(impact Impact) bool {
	for _, change := range r.Changes {
		if change.Impact == impact {
			return true
		}
	}
	return false
}

type _Comparison struct {
	report *Report
}

func (c *_Comparison) compareDeclaration(baseline, candidate *Declaration) {
	if baseline.Kind != candidate.Kind {
		c.add(ImpactBreaking, "declaration.type.changed", candidate.SkelName,
			fmt.Sprintf("declaration type changed from %s to %s", baseline.Kind, candidate.Kind), baseline.Pos, candidate.Pos)
		return
	}
	if baseline.Pub != candidate.Pub {
		impact := ImpactCompatible
		code := "declaration.visibility.increased"
		message := "declaration became public"
		if baseline.Pub {
			impact = ImpactBreaking
			code = "declaration.visibility.reduced"
			message = "public declaration became non-public"
		}
		c.add(impact, code, candidate.SkelName, message, baseline.Pos, candidate.Pos)
	}
	c.compareMetadata("declaration", candidate.SkelName, baseline.Metadata, candidate.Metadata, baseline.Pos, candidate.Pos)
	switch baseline.Kind {
	case "enum":
		c.compareEnum(candidate.SkelName, baseline.Enum, candidate.Enum)
	case "data", "config", "event":
		c.compareData(candidate.SkelName, baseline.Data, candidate.Data)
	case "actor":
		c.compareActor(candidate.SkelName, baseline.Actor, candidate.Actor)
	case "resource":
		c.compareResource(candidate.SkelName, baseline.Resource, candidate.Resource)
	case "service":
		c.compareService(candidate.SkelName, baseline.Service, candidate.Service)
	case "web":
		c.compareAudiences(candidate.SkelName, "web.audience", baseline.Web.Audiences, candidate.Web.Audiences)
	case "task":
		c.compareTask(candidate.SkelName, baseline.Task, candidate.Task)
	}
}

func (c *_Comparison) compareEnum(owner string, baseline, candidate *EnumSchema) {
	baselineByName := enumItemsByName(baseline.Items)
	candidateByName := enumItemsByName(candidate.Items)
	for _, item := range baseline.Items {
		other := candidateByName[item.Name]
		symbol := owner + "." + item.Name
		if other == nil {
			c.add(ImpactBreaking, "enum.item.removed", symbol, fmt.Sprintf("enum item %s was removed", item.Name), item.Pos, model.Position{})
			continue
		}
		c.compareMetadata("enum.item", symbol, item.Metadata, other.Metadata, item.Pos, other.Pos)
	}
	for _, item := range candidate.Items {
		if baselineByName[item.Name] == nil {
			c.add(ImpactDangerous, "enum.item.added", owner+"."+item.Name,
				fmt.Sprintf("enum item %s was added", item.Name), model.Position{}, item.Pos)
		}
	}
}

func (c *_Comparison) compareData(owner string, baseline, candidate *DataSchema) {
	if baseline.Lifecycle != candidate.Lifecycle {
		c.add(ImpactBreaking, "config.lifecycle.changed", owner,
			fmt.Sprintf("config lifecycle changed from %s to %s", baseline.Lifecycle, candidate.Lifecycle), model.Position{}, model.Position{})
	}
	if baseline.Sensitive != candidate.Sensitive {
		c.add(ImpactDangerous, "data.sensitive.changed", owner, "data sensitivity changed", model.Position{}, model.Position{})
	}
	if !slices.Equal(baseline.TypeParameters, candidate.TypeParameters) {
		c.add(ImpactBreaking, "data.type-parameters.changed", owner, "data type parameters changed", model.Position{}, model.Position{})
	}
	c.compareMembers(owner, "data.member", baseline.Members, candidate.Members, ImpactDangerous)
}

func (c *_Comparison) compareMembers(owner, prefix string, baseline, candidate []*Member, reorderImpact Impact) {
	baselineByName := membersByName(baseline)
	candidateByName := membersByName(candidate)
	for _, member := range baseline {
		other := candidateByName[member.Name]
		symbol := owner + "." + member.Name
		if other == nil {
			c.add(ImpactBreaking, prefix+".removed", symbol, fmt.Sprintf("member %s was removed", member.Name), member.Pos, model.Position{})
			continue
		}
		if !reflect.DeepEqual(member.Type, other.Type) {
			c.add(ImpactBreaking, prefix+".type.changed", symbol,
				fmt.Sprintf("member type changed from %s to %s", typeDisplay(member.Type), typeDisplay(other.Type)), member.Pos, other.Pos)
		}
		if member.Sensitive != other.Sensitive {
			c.add(ImpactDangerous, prefix+".sensitive.changed", symbol, "member sensitivity changed", member.Pos, other.Pos)
		}
		if member.Example != other.Example {
			c.add(ImpactCompatible, prefix+".example.changed", symbol, "member example changed", member.Pos, other.Pos)
		}
		c.compareMetadata(prefix, symbol, member.Metadata, other.Metadata, member.Pos, other.Pos)
	}
	for _, member := range candidate {
		if baselineByName[member.Name] == nil {
			c.add(ImpactBreaking, prefix+".added", owner+"."+member.Name,
				fmt.Sprintf("required member %s was added", member.Name), model.Position{}, member.Pos)
		}
	}
	if sameNamedSet(memberNames(baseline), memberNames(candidate)) && !slices.Equal(memberNames(baseline), memberNames(candidate)) {
		c.add(reorderImpact, prefix+".order.changed", owner, "member order changed", model.Position{}, model.Position{})
	}
}

func (c *_Comparison) compareActor(owner string, baseline, candidate *ActorSchema) {
	c.compareStringSet(owner, "actor.via", actorViaNames(baseline.Vias), actorViaNames(candidate.Vias), ImpactBreaking, ImpactCompatible)
	if baseline.AuthEnabled != candidate.AuthEnabled {
		impact := ImpactCompatible
		code := "actor.auth.added"
		message := "actor authentication was added"
		if baseline.AuthEnabled {
			impact, code, message = ImpactBreaking, "actor.auth.removed", "actor authentication was removed"
		}
		c.add(impact, code, owner, message, model.Position{}, model.Position{})
	}
	if baseline.AuthEnabled && candidate.AuthEnabled {
		c.compareMembers(owner+".credential", "actor.auth-credential.member", baseline.AuthCredential.Members, candidate.AuthCredential.Members, ImpactDangerous)
		c.compareMembers(owner+".info", "actor.auth-info.member", baseline.AuthInfo.Members, candidate.AuthInfo.Members, ImpactDangerous)
	}
	if baseline.PermEnabled != candidate.PermEnabled {
		impact := ImpactCompatible
		code := "actor.permission.added"
		message := "actor permission support was added"
		if baseline.PermEnabled {
			impact, code, message = ImpactBreaking, "actor.permission.removed", "actor permission support was removed"
		}
		c.add(impact, code, owner, message, model.Position{}, model.Position{})
	}
}

func (c *_Comparison) compareResource(owner string, baseline, candidate *ResourceSchema) {
	c.compareResourceChecks(owner, baseline.Checks, candidate.Checks)
	baselineByName := resourceActionsByName(baseline.Actions)
	candidateByName := resourceActionsByName(candidate.Actions)
	for _, action := range baseline.Actions {
		other := candidateByName[action.Name]
		symbol := owner + "." + action.Name
		if other == nil {
			c.add(ImpactBreaking, "resource.action.removed", symbol, fmt.Sprintf("resource action %s was removed", action.Name), action.Pos, model.Position{})
			continue
		}
		if action.PermissionCode != other.PermissionCode {
			c.add(ImpactBreaking, "resource.action.code.changed", symbol, "resource action permission code changed", action.Pos, other.Pos)
		}
		c.compareMetadata("resource.action", symbol, action.Metadata, other.Metadata, action.Pos, other.Pos)
		c.compareResourceChecks(symbol, action.Checks, other.Checks)
	}
	for _, action := range candidate.Actions {
		if baselineByName[action.Name] == nil {
			c.add(ImpactCompatible, "resource.action.added", owner+"."+action.Name,
				fmt.Sprintf("resource action %s was added", action.Name), model.Position{}, action.Pos)
		}
	}
}

func (c *_Comparison) compareResourceChecks(owner string, baseline, candidate []*ResourceCheck) {
	baselineByName := resourceChecksByName(baseline)
	candidateByName := resourceChecksByName(candidate)
	for _, check := range baseline {
		other := candidateByName[check.Name]
		symbol := owner + "." + check.Name
		if other == nil {
			c.add(ImpactBreaking, "resource.check.removed", symbol, fmt.Sprintf("resource check %s was removed", check.Name), check.Pos, model.Position{})
			continue
		}
		c.compareArguments(symbol, "resource.check.argument", check.Arguments, other.Arguments)
		c.compareMetadata("resource.check", symbol, check.Metadata, other.Metadata, check.Pos, other.Pos)
	}
	for _, check := range candidate {
		if baselineByName[check.Name] == nil {
			c.add(ImpactCompatible, "resource.check.added", owner+"."+check.Name,
				fmt.Sprintf("resource check %s was added", check.Name), model.Position{}, check.Pos)
		}
	}
}

func (c *_Comparison) compareService(owner string, baseline, candidate *ServiceSchema) {
	c.compareAudiences(owner, "service.audience", baseline.Audiences, candidate.Audiences)
	c.compareAuth(owner, "service", baseline.Auth, candidate.Auth, model.Position{}, model.Position{})
	c.compareRequirement(owner, "service", baseline.Require, candidate.Require, model.Position{}, model.Position{})
	baselineByName := methodsByName(baseline.Methods)
	candidateByName := methodsByName(candidate.Methods)
	for _, method := range baseline.Methods {
		other := candidateByName[method.Name]
		symbol := owner + "." + method.Name
		if other == nil {
			c.add(ImpactBreaking, "service.method.removed", symbol, fmt.Sprintf("service method %s was removed", method.Name), method.Pos, model.Position{})
			continue
		}
		c.compareMethod(symbol, method, other)
	}
	for _, method := range candidate.Methods {
		if baselineByName[method.Name] == nil {
			c.add(ImpactCompatible, "service.method.added", owner+"."+method.Name,
				fmt.Sprintf("service method %s was added", method.Name), model.Position{}, method.Pos)
		}
	}
}

func (c *_Comparison) compareMethod(owner string, baseline, candidate *Method) {
	c.compareAuth(owner, "method", baseline.Auth, candidate.Auth, baseline.Pos, candidate.Pos)
	c.compareRequirement(owner, "method", baseline.Require, candidate.Require, baseline.Pos, candidate.Pos)
	c.compareArguments(owner, "method.argument", baseline.Arguments, candidate.Arguments)
	if !reflect.DeepEqual(baseline.Result, candidate.Result) {
		c.add(ImpactBreaking, "method.result.changed", owner,
			fmt.Sprintf("method result changed from %s to %s", typeDisplay(baseline.Result), typeDisplay(candidate.Result)), baseline.Pos, candidate.Pos)
	}
	if baseline.ArgumentsSensitive != candidate.ArgumentsSensitive || baseline.ResultSensitive != candidate.ResultSensitive {
		c.add(ImpactDangerous, "method.sensitive.changed", owner, "method sensitivity changed", baseline.Pos, candidate.Pos)
	}
	if baseline.Example != candidate.Example || baseline.InputDescription != candidate.InputDescription ||
		baseline.OutputDescription != candidate.OutputDescription || baseline.OutputExample != candidate.OutputExample {
		c.add(ImpactCompatible, "method.documentation.changed", owner, "method documentation or examples changed", baseline.Pos, candidate.Pos)
	}
	c.compareMetadata("method", owner, baseline.Metadata, candidate.Metadata, baseline.Pos, candidate.Pos)
}

func (c *_Comparison) compareArguments(owner, prefix string, baseline, candidate []*Argument) {
	baselineByName := argumentsByName(baseline)
	candidateByName := argumentsByName(candidate)
	for _, argument := range baseline {
		other := candidateByName[argument.Name]
		symbol := owner + "." + argument.Name
		if other == nil {
			c.add(ImpactBreaking, prefix+".removed", symbol, fmt.Sprintf("argument %s was removed", argument.Name), argument.Pos, model.Position{})
			continue
		}
		if !reflect.DeepEqual(argument.Type, other.Type) {
			c.add(ImpactBreaking, prefix+".type.changed", symbol,
				fmt.Sprintf("argument type changed from %s to %s", typeDisplay(argument.Type), typeDisplay(other.Type)), argument.Pos, other.Pos)
		}
		if argument.Sensitive != other.Sensitive {
			c.add(ImpactDangerous, prefix+".sensitive.changed", symbol, "argument sensitivity changed", argument.Pos, other.Pos)
		}
		if argument.Example != other.Example {
			c.add(ImpactCompatible, prefix+".example.changed", symbol, "argument example changed", argument.Pos, other.Pos)
		}
		c.compareMetadata(prefix, symbol, argument.Metadata, other.Metadata, argument.Pos, other.Pos)
	}
	for _, argument := range candidate {
		if baselineByName[argument.Name] == nil {
			c.add(ImpactBreaking, prefix+".added", owner+"."+argument.Name,
				fmt.Sprintf("required argument %s was added", argument.Name), model.Position{}, argument.Pos)
		}
	}
	if sameNamedSet(argumentNames(baseline), argumentNames(candidate)) && !slices.Equal(argumentNames(baseline), argumentNames(candidate)) {
		c.add(ImpactBreaking, prefix+".order.changed", owner, "argument order changed", model.Position{}, model.Position{})
	}
}

func (c *_Comparison) compareTask(owner string, baseline, candidate *TaskSchema) {
	baselineByName := triggersByName(baseline.Triggers)
	candidateByName := triggersByName(candidate.Triggers)
	for _, trigger := range baseline.Triggers {
		other := candidateByName[trigger.Name]
		symbol := owner + "." + trigger.Name
		if other == nil {
			c.add(ImpactBreaking, "task.trigger.removed", symbol, fmt.Sprintf("task trigger %s was removed", trigger.Name), trigger.Pos, model.Position{})
			continue
		}
		c.compareArguments(symbol, "task.trigger.argument", trigger.Arguments, other.Arguments)
		if trigger.ArgumentsSensitive != other.ArgumentsSensitive {
			c.add(ImpactDangerous, "task.trigger.sensitive.changed", symbol, "task trigger sensitivity changed", trigger.Pos, other.Pos)
		}
		if trigger.InputDescription != other.InputDescription {
			c.add(ImpactCompatible, "task.trigger.documentation.changed", symbol, "task trigger documentation changed", trigger.Pos, other.Pos)
		}
		c.compareMetadata("task.trigger", symbol, trigger.Metadata, other.Metadata, trigger.Pos, other.Pos)
	}
	for _, trigger := range candidate.Triggers {
		if baselineByName[trigger.Name] == nil {
			c.add(ImpactCompatible, "task.trigger.added", owner+"."+trigger.Name,
				fmt.Sprintf("task trigger %s was added", trigger.Name), model.Position{}, trigger.Pos)
		}
	}
}

func (c *_Comparison) compareAudiences(owner, prefix string, baseline, candidate []*Audience) {
	baselineByKey := audiencesByKey(baseline)
	candidateByKey := audiencesByKey(candidate)
	for key, audience := range baselineByKey {
		if candidateByKey[key] == nil {
			c.add(ImpactBreaking, prefix+".removed", owner,
				fmt.Sprintf("audience %s via %s was removed", audience.Actor, audience.Via), audience.Pos, model.Position{})
		}
	}
	for key, audience := range candidateByKey {
		if baselineByKey[key] == nil {
			c.add(ImpactCompatible, prefix+".added", owner,
				fmt.Sprintf("audience %s via %s was added", audience.Actor, audience.Via), model.Position{}, audience.Pos)
		}
	}
}

func (c *_Comparison) compareAuth(owner, prefix, baseline, candidate string, baselinePos, candidatePos model.Position) {
	if baseline == candidate {
		return
	}
	impact := ImpactDangerous
	code := prefix + ".auth.changed"
	if candidate == string(model.AuthModeAuth) {
		impact, code = ImpactBreaking, prefix+".auth.tightened"
	} else if baseline == string(model.AuthModeAuth) {
		code = prefix + ".auth.relaxed"
	}
	c.add(impact, code, owner, fmt.Sprintf("authentication changed from %s to %s", baseline, candidate), baselinePos, candidatePos)
}

func (c *_Comparison) compareRequirement(owner, prefix string, baseline, candidate *Requirement, baselinePos, candidatePos model.Position) {
	if reflect.DeepEqual(baseline, candidate) {
		return
	}
	impact := ImpactBreaking
	code := prefix + ".require.changed"
	message := "permission requirement changed"
	if baseline == nil {
		code, message = prefix+".require.added", "permission requirement was added"
	} else if candidate == nil {
		impact, code, message = ImpactDangerous, prefix+".require.removed", "permission requirement was removed"
	}
	c.add(impact, code, owner, message, baselinePos, candidatePos)
}

func (c *_Comparison) compareMetadata(prefix, symbol string, baseline, candidate Metadata, baselinePos, candidatePos model.Position) {
	if baseline.Description != candidate.Description {
		c.add(ImpactCompatible, prefix+".description.changed", symbol, "description changed", baselinePos, candidatePos)
	}
	if baseline.Deprecated != candidate.Deprecated {
		message := "deprecation was removed"
		if candidate.Deprecated {
			message = "deprecation was added"
		}
		c.add(ImpactCompatible, prefix+".deprecated.changed", symbol, message, baselinePos, candidatePos)
	}
	if baseline.DeprecatedReason != candidate.DeprecatedReason {
		c.add(ImpactCompatible, prefix+".deprecated-reason.changed", symbol, "deprecation reason changed", baselinePos, candidatePos)
	}
}

func (c *_Comparison) compareStringSet(owner, prefix string, baseline, candidate []string, removedImpact, addedImpact Impact) {
	baselineSet := stringSet(baseline)
	candidateSet := stringSet(candidate)
	for _, value := range baseline {
		if !candidateSet[value] {
			c.add(removedImpact, prefix+".removed", owner, fmt.Sprintf("%s %s was removed", prefix, value), model.Position{}, model.Position{})
		}
	}
	for _, value := range candidate {
		if !baselineSet[value] {
			c.add(addedImpact, prefix+".added", owner, fmt.Sprintf("%s %s was added", prefix, value), model.Position{}, model.Position{})
		}
	}
}

func (c *_Comparison) add(impact Impact, code, symbol, message string, baseline, candidate model.Position) {
	c.report.Changes = append(c.report.Changes, &Change{
		Code: code, Impact: impact, Symbol: symbol, Message: message,
		Baseline: positionPointer(baseline), Candidate: positionPointer(candidate),
	})
}

func (c *_Comparison) finish() {
	slices.SortFunc(c.report.Changes, func(left, right *Change) int {
		if order := impactOrder(left.Impact) - impactOrder(right.Impact); order != 0 {
			return order
		}
		if order := strings.Compare(left.Symbol, right.Symbol); order != 0 {
			return order
		}
		if order := strings.Compare(left.Code, right.Code); order != 0 {
			return order
		}
		return strings.Compare(left.Message, right.Message)
	})
	for _, change := range c.report.Changes {
		switch change.Impact {
		case ImpactBreaking:
			c.report.Summary.Breaking++
		case ImpactDangerous:
			c.report.Summary.Dangerous++
		case ImpactCompatible:
			c.report.Summary.Compatible++
		}
	}
	c.report.Compatible = c.report.Summary.Breaking == 0
}

func impactOrder(impact Impact) int {
	switch impact {
	case ImpactBreaking:
		return 1
	case ImpactDangerous:
		return 2
	case ImpactCompatible:
		return 3
	default:
		return 99
	}
}

func positionPointer(position model.Position) *model.Position {
	if position.File == "" && position.Line == 0 && position.Column == 0 {
		return nil
	}
	return new(model.Position(position))
}

func declarationsByKey(values []*Declaration) map[string]*Declaration {
	result := make(map[string]*Declaration, len(values))
	for _, value := range values {
		result[declarationKey(value)] = value
	}
	return result
}

func unmatchedDeclarationsByName(values []*Declaration, matched map[*Declaration]bool) map[string][]*Declaration {
	result := map[string][]*Declaration{}
	for _, value := range values {
		if !matched[value] {
			result[value.SkelName] = append(result[value.SkelName], value)
		}
	}
	return result
}

func declarationKey(value *Declaration) string {
	return value.Kind + "\x00" + value.SkelName
}

func enumItemsByName(values []*EnumItem) map[string]*EnumItem {
	result := make(map[string]*EnumItem, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func membersByName(values []*Member) map[string]*Member {
	result := make(map[string]*Member, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func methodsByName(values []*Method) map[string]*Method {
	result := make(map[string]*Method, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func argumentsByName(values []*Argument) map[string]*Argument {
	result := make(map[string]*Argument, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func resourceActionsByName(values []*ResourceAction) map[string]*ResourceAction {
	result := make(map[string]*ResourceAction, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func resourceChecksByName(values []*ResourceCheck) map[string]*ResourceCheck {
	result := make(map[string]*ResourceCheck, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func triggersByName(values []*Trigger) map[string]*Trigger {
	result := make(map[string]*Trigger, len(values))
	for _, value := range values {
		result[value.Name] = value
	}
	return result
}

func audiencesByKey(values []*Audience) map[string]*Audience {
	result := make(map[string]*Audience, len(values))
	for _, value := range values {
		result[value.Actor+"\x00"+value.Via] = value
	}
	return result
}

func actorViaNames(values []*ActorVia) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Name)
	}
	return result
}

func memberNames(values []*Member) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Name)
	}
	return result
}

func argumentNames(values []*Argument) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.Name)
	}
	return result
}

func sameNamedSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftSet := stringSet(left)
	for _, value := range right {
		if !leftSet[value] {
			return false
		}
	}
	return true
}

func stringSet(values []string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func typeDisplay(value *Type) string {
	if value == nil {
		return "void"
	}
	var result string
	switch value.Kind {
	case "list":
		result = "list<" + typeDisplay(value.Element) + ">"
	case "map":
		result = "map<" + typeDisplay(value.Key) + ", " + typeDisplay(value.Value) + ">"
	case "data", "reference":
		result = value.Name
		if len(value.Arguments) > 0 {
			arguments := make([]string, 0, len(value.Arguments))
			for _, argument := range value.Arguments {
				arguments = append(arguments, typeDisplay(argument))
			}
			result += "<" + strings.Join(arguments, ", ") + ">"
		}
	case "enum", "scalar", "typeParameter":
		result = value.Name
	default:
		result = value.Kind
	}
	if value.Nullable {
		result += "?"
	}
	return result
}
