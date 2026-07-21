package source

import (
	"testing"

	"go.yorun.ai/skelc/model"
)

func TestCastEnum(t *testing.T) {
	enum := castEnum(&model.Enum{
		Name:        "UserStatus",
		Description: "User status",
		UnspecifiedItem: &model.EnumItem{
			Name: "UNSPECIFIED",
		},
		Items: []*model.EnumItem{
			{
				Name:        "ACTIVE",
				Description: "Active",
			},
			{
				Name: "DISABLED",
			},
		},
	})

	if enum.Name != "UserStatus" {
		t.Fatalf("unexpected enum name: %s", enum.Name)
	}
	if enum.VarName != "userStatus" {
		t.Fatalf("unexpected enum var name: %s", enum.VarName)
	}
	if len(enum.CommentLines) == 0 || enum.CommentLines[0] != "UserStatus User status" {
		t.Fatalf("unexpected enum comment lines: %+v", enum.CommentLines)
	}
	if enum.UnspecifiedItem.Name != "UserStatusUnspecified" {
		t.Fatalf("unexpected unspecified item name: %s", enum.UnspecifiedItem.Name)
	}
	if enum.Items[0].Name != "UserStatusActive" {
		t.Fatalf("unexpected first item name: %s", enum.Items[0].Name)
	}
	if enum.Items[0].Value != "ACTIVE" {
		t.Fatalf("unexpected first item value: %s", enum.Items[0].Value)
	}
	if len(enum.Items[0].CommentLines) == 0 || enum.Items[0].CommentLines[0] != "UserStatusActive Active" {
		t.Fatalf("unexpected first item comment lines: %+v", enum.Items[0].CommentLines)
	}
}
