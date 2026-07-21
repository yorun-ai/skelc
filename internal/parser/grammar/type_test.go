package grammar

import "testing"

func TestParsePlainTypes(t *testing.T) {
	for _, plainType := range PlainTypes {
		content := parseSkelForTest(t, "domain demo.user\ndata Sample { value: "+string(plainType)+" }")
		got := content.Entries[0].Data.Members[0].Type.Plain
		if got == nil || *got != plainType {
			t.Fatalf("plain type %q parsed as %+v", plainType, got)
		}
	}
}
