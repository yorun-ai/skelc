package webpath

import "testing"

func TestValidate(t *testing.T) {
	for _, value := range []string{"/", "/portal", "/portal/v1-assets/", "/.well-known", "/用户"} {
		if err := Validate(value); err != nil {
			t.Errorf("valid path %q: %v", value, err)
		}
	}
	for _, value := range []string{"", "relative", "//host", "/a//b", "/./a", "/a/..", "/a?b", "/a#b", "/a b", "/a\nb", "/a\\b", "/:id", "/a/*", "/{id}", "/a%2fb"} {
		if err := Validate(value); err == nil {
			t.Errorf("accepted invalid path %q", value)
		}
	}
}
