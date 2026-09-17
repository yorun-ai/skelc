package skelmeta

import "testing"

func TestSensitiveMarkerFieldNameMatchesGeneratedMethod(t *testing.T) {
	if SensitiveMarkerMethodName != "SkelSensitive" {
		t.Fatalf("unexpected marker method name: %q", SensitiveMarkerMethodName)
	}
	if got := SensitiveMarkerFieldName(); got != "skelSensitive" {
		t.Fatalf("SensitiveMarkerFieldName() = %q; want %q", got, "skelSensitive")
	}
}
