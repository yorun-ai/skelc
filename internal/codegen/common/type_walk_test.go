package common

import (
	"errors"
	"reflect"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestWalkTypeVisitsStructuralChildrenInOrder(t *testing.T) {
	key := &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarString}
	value := &model.Type{Kind: model.TypeKindScalar, Scalar: model.ScalarInt}
	root := &model.Type{Kind: model.TypeKindMap, Map: &model.MapType{Key: key, Value: &model.Type{
		Kind: model.TypeKindList, List: &model.ListType{Value: value},
	}}}
	kinds := []model.TypeKind{}
	if err := WalkType(root, func(type_ *model.Type) error {
		kinds = append(kinds, type_.Kind)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	expected := []model.TypeKind{model.TypeKindMap, model.TypeKindScalar, model.TypeKindList, model.TypeKindScalar}
	if !reflect.DeepEqual(kinds, expected) {
		t.Fatalf("unexpected walk order: %v", kinds)
	}
}

func TestWalkTypeGraphTerminatesOnRecursiveData(t *testing.T) {
	data := &model.Data{Name: "Node", Kind: model.DataKindData}
	reference := &model.Type{Kind: model.TypeKindData, Data: data}
	data.Members = []*model.DataMember{{Name: "next", Type: reference}}
	visits := 0
	if err := WalkTypeGraph(reference, func(*model.Type) error {
		visits++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if visits != 1 {
		t.Fatalf("expected recursive type to be visited once, got %d", visits)
	}
}

func TestWalkTypePropagatesVisitorError(t *testing.T) {
	expected := errors.New("stop")
	err := WalkType(&model.Type{Kind: model.TypeKindScalar}, func(*model.Type) error { return expected })
	if !errors.Is(err, expected) {
		t.Fatalf("expected visitor error, got %v", err)
	}
}
