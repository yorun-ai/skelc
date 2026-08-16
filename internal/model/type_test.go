package model

import (
	"fmt"
	"testing"
)

func ExampleType_ContainsBinaryType() {
	attachment := &Data{
		Name: "Attachment",
		Members: []*DataMember{
			{Name: "content", Type: &Type{Kind: TypeKindScalar, Scalar: ScalarBinary}},
		},
	}
	attachmentType := &Type{Kind: TypeKindData, Data: attachment}

	fmt.Println(attachmentType.Name())
	fmt.Println(attachmentType.ContainsBinaryType())

	// Output:
	// Attachment
	// true
}

func TestTypeContainsBinaryType(t *testing.T) {
	binaryData := &Data{
		Name: "Attachment",
		Members: []*DataMember{
			{Name: "content", Type: &Type{Kind: TypeKindScalar, Scalar: ScalarBinary}},
		},
	}
	valueType := &Type{Kind: TypeKindData, Data: binaryData}

	if !valueType.ContainsBinaryType() {
		t.Fatal("expected data containing binary member to report binary type")
	}
	if valueType.Name() != "Attachment" {
		t.Fatalf("unexpected type name: %q", valueType.Name())
	}
}
