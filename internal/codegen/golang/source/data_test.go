package source

import (
	"reflect"
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestCastData(t *testing.T) {
	data := castData(&model.Data{
		Name:        "Page",
		Description: "Paginated result",
		TypeParameters: []*model.TypeParameter{
			{Name: "TItem"},
		},
		Members: []*model.DataMember{
			{
				Name:        "generatedAt",
				Description: "Generated at",
				Type: &model.Type{
					Kind:   model.TypeKindScalar,
					Scalar: model.ScalarTimestamp,
				},
			},
			{
				Name:        "avatarUrl",
				Description: "Avatar URL",
				Example:     `"https://xxx.com/a.png"`,
				Type: &model.Type{
					Kind:     model.TypeKindScalar,
					Scalar:   model.ScalarString,
					Nullable: true,
				},
			},
		},
	})

	if data.FullName != "Page[TItem any]" {
		t.Fatalf("unexpected full name: %s", data.FullName)
	}
	if len(data.CommentLines) == 0 || data.CommentLines[0] != "Page Paginated result" {
		t.Fatalf("unexpected data comment lines: %+v", data.CommentLines)
	}
	if len(data.Members) != 2 {
		t.Fatalf("unexpected member count: %d", len(data.Members))
	}
	if data.Members[0].Type.Plain != "skel.Timestamp" {
		t.Fatalf("unexpected first member type: %s", data.Members[0].Type.Plain)
	}
	if len(data.Members[1].CommentLines) == 0 || data.Members[1].CommentLines[0] != `AvatarUrl Avatar URL (e.g. "https://xxx.com/a.png")` {
		t.Fatalf("unexpected second member comment lines: %+v", data.Members[1].CommentLines)
	}
}

func TestBuildDataImports(t *testing.T) {
	imports := buildDataImports([]*Data{
		{
			Members: []*DataMember{
				{Type: &Type{Imports: []*Import{{Path: skelImport}}}},
				{Type: &Type{Imports: []*Import{{Path: skelImport}}}},
			},
		},
	})

	if got, want := importPaths(imports), []string{skelImport}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected import paths: got=%v want=%v", got, want)
	}
}

func TestCastDataMapsDurationToSkelDuration(t *testing.T) {
	data := castData(&model.Data{
		Name: "TimeoutConfig",
		Members: []*model.DataMember{
			{
				Name: "timeout",
				Type: &model.Type{
					Kind:   model.TypeKindScalar,
					Scalar: model.ScalarDuration,
				},
			},
		},
	})
	if data.Members[0].Type.Plain != "skel.Duration" {
		t.Fatalf("unexpected duration member type: %s", data.Members[0].Type.Plain)
	}
}

func TestCastDataMapsLocalDateToSkelLocalDate(t *testing.T) {
	data := castData(&model.Data{
		Name: "Profile",
		Members: []*model.DataMember{
			{
				Name: "birthday",
				Type: &model.Type{
					Kind:   model.TypeKindScalar,
					Scalar: model.ScalarLocalDate,
				},
			},
		},
	})
	if data.Members[0].Type.Plain != "skel.LocalDate" {
		t.Fatalf("unexpected date member type: %s", data.Members[0].Type.Plain)
	}
}
