package analyzer

import "testing"

func TestMatchesCase(t *testing.T) {
	cases := []struct {
		value     string
		_CaseType _CaseType
		want      bool
	}{
		{value: "user_name", _CaseType: caseTypeSnake, want: true},
		{value: "USER_NAME", _CaseType: caseTypeScreamingSnake, want: true},
		{value: "UserName", _CaseType: caseTypeCamel, want: true},
		{value: "userName", _CaseType: caseTypeLowerCamel, want: true},
		{value: "UserName", _CaseType: caseTypeLowerCamel, want: false},
	}

	for _, tc := range cases {
		if got := matchesCase(tc.value, tc._CaseType); got != tc.want {
			t.Fatalf("matchesCase(%q, %q) = %v, want %v", tc.value, tc._CaseType, got, tc.want)
		}
	}
}

func TestConvertCase(t *testing.T) {
	tests := []struct {
		value     string
		_CaseType _CaseType
		want      string
	}{
		{value: "UserName", _CaseType: caseTypeSnake, want: "user_name"},
		{value: "userName", _CaseType: caseTypeScreamingSnake, want: "USER_NAME"},
		{value: "user_name", _CaseType: caseTypeCamel, want: "UserName"},
		{value: "UserName", _CaseType: caseTypeLowerCamel, want: "userName"},
		{value: "unchanged", _CaseType: _CaseType("custom"), want: "unchanged"},
	}

	for _, test := range tests {
		if got := convertCase(test.value, test._CaseType); got != test.want {
			t.Errorf("convertCase(%q, %q) = %q, want %q", test.value, test._CaseType, got, test.want)
		}
	}
}
