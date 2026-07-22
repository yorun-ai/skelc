package schema

import (
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/model"
)

func mustView(t *testing.T, mode view.Mode, domain *model.Domain) *view.Domain {
	t.Helper()
	result, err := view.New(mode, domain)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
