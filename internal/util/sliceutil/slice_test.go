package sliceutil

import (
	"reflect"
	"testing"
)

func TestMapAndFilterPreserveOrder(t *testing.T) {
	mapped := Map([]int{1, 2, 3}, func(value int) int { return value * 2 })
	filtered := Filter(mapped, func(value int) bool { return value > 2 })
	if !reflect.DeepEqual(filtered, []int{4, 6}) {
		t.Fatalf("unexpected result: %v", filtered)
	}
}

func TestFind(t *testing.T) {
	value, ok := Find([]string{"a", "bb"}, func(value string) bool { return len(value) == 2 })
	if !ok || value != "bb" {
		t.Fatalf("unexpected result: %q, %v", value, ok)
	}
}
