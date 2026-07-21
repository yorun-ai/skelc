package grammar

import "testing"

func TestUnquoteDescriptionString(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		want      string
		wantPanic bool
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
			wantPanic: true,
		},
		{
			name:      "single quoted rejected",
			raw:       `'User domain'`,
			wantPanic: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.wantPanic {
				defer func() {
					if recover() == nil {
						t.Fatal("expected panic")
					}
				}()
				_ = UnquoteDescriptionString(test.raw)
				return
			}
			got := UnquoteDescriptionString(test.raw)
			if got != test.want {
				t.Fatalf("unexpected value: got=%q want=%q", got, test.want)
			}
		})
	}
}
