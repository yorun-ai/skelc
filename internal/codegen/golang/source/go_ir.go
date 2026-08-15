package source

var goIRTemplate = loadTemplate("go_ir.go.tpl")

type _GoParameter struct {
	Name string
	Type string
}

type _GoBlock struct {
	Statements []*_GoStatement
}

type _GoStatement struct {
	Assignment *_GoAssignment
	Expression *_GoExpression
	If         *_GoIf
	Range      *_GoRange
	Return     *_GoReturn
}

type _GoAssignment struct {
	Targets  []string
	Operator string
	Values   []*_GoExpression
}

type _GoIf struct {
	Init      *_GoAssignment
	Condition *_GoExpression
	Then      *_GoBlock
	Else      *_GoBlock
}

type _GoRange struct {
	Names  []string
	Source *_GoExpression
	Body   *_GoBlock
}

type _GoReturn struct {
	Values []*_GoExpression
}

type _GoExpression struct {
	Raw      string
	Call     *_GoCall
	Function *_GoFunction
}

type _GoCall struct {
	Function  *_GoExpression
	Arguments []*_GoExpression
}

type _GoFunction struct {
	Parameters []*_GoParameter
	Result     string
	Body       *_GoBlock
}

func goParameter(name string, type_ string) *_GoParameter {
	return &_GoParameter{Name: name, Type: type_}
}

func goBlock(statements ...*_GoStatement) *_GoBlock {
	block := &_GoBlock{}
	block.append(statements...)
	return block
}

func (b *_GoBlock) append(statements ...*_GoStatement) {
	for _, statement := range statements {
		if statement != nil {
			b.Statements = append(b.Statements, statement)
		}
	}
}

func goRaw(raw string) *_GoExpression {
	return &_GoExpression{Raw: raw}
}

func goCall(function string, arguments ...*_GoExpression) *_GoExpression {
	return goCallExpression(goRaw(function), arguments...)
}

func goCallExpression(function *_GoExpression, arguments ...*_GoExpression) *_GoExpression {
	return &_GoExpression{Call: &_GoCall{Function: function, Arguments: arguments}}
}

func goFunction(parameters []*_GoParameter, result string, body *_GoBlock) *_GoFunction {
	return &_GoFunction{Parameters: parameters, Result: result, Body: body}
}

func goFunctionExpression(function *_GoFunction) *_GoExpression {
	return &_GoExpression{Function: function}
}

func goAssignment(target string, operator string, value *_GoExpression) *_GoAssignment {
	return &_GoAssignment{Targets: []string{target}, Operator: operator, Values: []*_GoExpression{value}}
}

func goAssignmentStatement(target string, operator string, value *_GoExpression) *_GoStatement {
	return &_GoStatement{Assignment: goAssignment(target, operator, value)}
}

func goExpressionStatement(expression *_GoExpression) *_GoStatement {
	return &_GoStatement{Expression: expression}
}

func goIfStatement(init *_GoAssignment, condition *_GoExpression, thenBlock *_GoBlock, elseBlock *_GoBlock) *_GoStatement {
	return &_GoStatement{If: &_GoIf{Init: init, Condition: condition, Then: thenBlock, Else: elseBlock}}
}

func goRangeStatement(names []string, source *_GoExpression, body *_GoBlock) *_GoStatement {
	return &_GoStatement{Range: &_GoRange{Names: names, Source: source, Body: body}}
}

func goReturnStatement(values ...*_GoExpression) *_GoStatement {
	return &_GoStatement{Return: &_GoReturn{Values: values}}
}
