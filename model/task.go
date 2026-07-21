package model

// Task describes a background task declaration.
type Task struct {
	// Pos is the task declaration's source position.
	Pos Position
	// Name is the task's local name.
	Name string
	// SkelName is the task's fully qualified Skel name.
	SkelName string
	// Hash is the task's compatibility hash.
	Hash string
	// Description is the task's documentation text.
	Description string
	// Triggers lists task triggers in source order.
	Triggers []*TaskTrigger
}

// TaskTrigger describes one way to invoke a task.
type TaskTrigger struct {
	// Pos is the trigger declaration's source position.
	Pos Position
	// Name is the trigger's normalized local name.
	Name string
	// SkelName is the trigger name as represented in Skel metadata.
	SkelName string
	// Hash is the trigger's compatibility hash.
	Hash string
	// Description is the trigger's documentation text.
	Description string
	// Example is the trigger's example text.
	Example string
	// InputDescription documents the trigger input as a whole.
	InputDescription string
	// Arguments lists trigger arguments in source order.
	Arguments []*Argument
	// ArgumentsData is the generated data model representing trigger arguments.
	ArgumentsData *Data
}
