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

func TestConvertCase(t *testing.T) {
	tests := []struct {
		value    string
		caseType caseType
		want     string
	}{
		{value: "UserName", caseType: caseTypeSnake, want: "user_name"},
		{value: "userName", caseType: caseTypeScreamingSnake, want: "USER_NAME"},
		{value: "user_name", caseType: caseTypeCamel, want: "UserName"},
		{value: "UserName", caseType: caseTypeLowerCamel, want: "userName"},
		{value: "unchanged", caseType: caseType("custom"), want: "unchanged"},
	}

	for _, test := range tests {
		if got := convertCase(test.value, test.caseType); got != test.want {
			t.Errorf("convertCase(%q, %q) = %q, want %q", test.value, test.caseType, got, test.want)
		}
	}
}
