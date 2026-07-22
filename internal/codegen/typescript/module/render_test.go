package module

import (
	"testing"

	"go.yorun.ai/skelc/internal/codegen/common"
)

func renderTemplate(t *testing.T, template string, payload any) string {
	t.Helper()
	content, err := common.RenderTemplate(template, payload)
	if err != nil {
		t.Fatal(err)
	}
	return content
}
