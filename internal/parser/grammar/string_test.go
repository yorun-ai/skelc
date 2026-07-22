package grammar

import "testing"

func TestUnquoteDescriptionString(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		want      string
		wantError bool
	}{
		{
			name: "single-line double quoted",
			raw:  `"User domain"`,
			want: "User domain",
		},
		{
			name: "triple double quoted",
			raw:  "\"\"\"\nUser domain\nSecond line description\n\"\"\"",
			want: "User domain\nSecond line description",
		},
		{
			name: "triple double quoted dedents common indentation",
			raw:  "\"\"\"\n    User domain\n    Second line description\n\"\"\"",
			want: "User domain\nSecond line description",
		},
		{
			name: "triple double quoted ignores blank lines while dedenting",
			raw:  "\"\"\"\n    User domain\n\n    Second line description\n\"\"\"",
			want: "User domain\n\nSecond line description",
		},
		{
			name: "triple double quoted keeps relative indentation",
			raw:  "\"\"\"\n    User domain\n        Second line description\n\"\"\"",
			want: "User domain\n    Second line description",
		},
		{
			name:      "triple double quoted same line rejected",
			raw:       "\"\"\"User domain\nSecond line description\"\"\"",
			wantError: true,
		},
		{
			name:      "single quoted rejected",
			raw:       `'User domain'`,
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := UnquoteDescriptionString(test.raw)
			if test.wantError {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("unexpected value: got=%q want=%q", got, test.want)
			}
		})
	}
}
