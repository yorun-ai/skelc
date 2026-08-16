package vineschema

import (
	"testing"

	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/internal/model"
)

func mustView(t *testing.T, mode view.Mode, domain *model.Domain) *view.Domain {
	t.Helper()
	result, err := view.New(mode, domain)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func mustBuildDomainSchema(t *testing.T, gen *_Gen) *_DomainSchema {
	t.Helper()
	result, err := gen.buildDomainSchema()
	if err != nil {
		t.Fatal(err)
	}
	return result
}
