package analyzer

import "testing"

func TestMatchesCase(t *testing.T) {
	cases := []struct {
		value    string
		caseType caseType
		want     bool
	}{
		{value: "user_name", caseType: caseTypeSnake, want: true},
		{value: "USER_NAME", caseType: caseTypeScreamingSnake, want: true},
		{value: "UserName", caseType: caseTypeCamel, want: true},
		{value: "userName", caseType: caseTypeLowerCamel, want: true},
		{value: "UserName", caseType: caseTypeLowerCamel, want: false},
	}

	for _, tc := range cases {
		if got := matchesCase(tc.value, tc.caseType); got != tc.want {
			t.Fatalf("matchesCase(%q, %q) = %v, want %v", tc.value, tc.caseType, got, tc.want)
		}
	}
}

func TestCaseTypeExample(t *testing.T) {
	if got := caseTypeExample(caseTypeCamel); got != "CamelCase" {
		t.Fatalf("unexpected case type example: %s", got)
	}
	if got := caseTypeExample(caseType("custom")); got != "custom" {
		t.Fatalf("unexpected fallback case type example: %s", got)
	}
}
