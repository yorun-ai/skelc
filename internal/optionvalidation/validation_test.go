package optionvalidation

import (
	"errors"
	"testing"
)

func TestNewValidationErrorCarriesFieldAndRule(t *testing.T) {
	err := NewValidationError(FieldGoVineVersion, RuleInvalid, "go-vine-version must be a semantic version")

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("NewValidationError must return a *ValidationError, got %T", err)
	}
	if validationErr.Field != FieldGoVineVersion || validationErr.Rule != RuleInvalid {
		t.Fatalf("unexpected field/rule: %+v", validationErr)
	}
	if err.Error() != "go-vine-version must be a semantic version" {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func TestValidationErrorKeepsFieldAndRuleIdentity(t *testing.T) {
	for _, test := range []struct {
		field Field
		rule  Rule
	}{
		{field: FieldSkelInput, rule: RuleRequired},
		{field: FieldGoPublicOutput, rule: RuleRequiresPublicOutput},
		{field: FieldSkeletonOutput, rule: RuleNoTrailingSlash},
	} {
		err := NewValidationError(test.field, test.rule, "invalid option").(*ValidationError)
		if err.Field != test.field || err.Rule != test.rule {
			t.Fatalf("unexpected validation error: %+v", err)
		}
	}
}
