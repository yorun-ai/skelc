package graphutil

import "testing"

func TestFindCycles(t *testing.T) {
	graph := New[string]()
	graph.AddEdge("a", "b")
	graph.AddEdge("b", "a")
	graph.AddEdge("c", "c")
	graph.AddEdge("d", "e")

	cycles := graph.FindCycles()
	if len(cycles) != 2 {
		t.Fatalf("expected two cycles, got %v", cycles)
	}
	if len(cycles[0]) != 2 || len(cycles[1]) != 1 || cycles[1][0] != "c" {
		t.Fatalf("unexpected cycles: %v", cycles)
	}
}
