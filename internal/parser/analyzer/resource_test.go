package analyzer

import "testing"

func TestResourceDefaultsToNonPub(t *testing.T) {
	content := parseResourceTestContent(t, `
domain demo

resource User {
    action read
}
`)
	domain := Analyze(content).Model()

	if len(domain.Resources()) != 1 {
		t.Fatalf("unexpected resource count: %d", len(domain.Resources()))
	}
	if domain.Resources()[0].Pub {
		t.Fatalf("resource Pub = true, want false")
	}
}

func TestPubResourceSetsPub(t *testing.T) {
	content := parseResourceTestContent(t, `
domain demo

pub resource User {
    action read
}
`)
	domain := Analyze(content).Model()

	if len(domain.Resources()) != 1 {
		t.Fatalf("unexpected resource count: %d", len(domain.Resources()))
	}
	if !domain.Resources()[0].Pub {
		t.Fatalf("resource Pub = false, want true")
	}
}

func TestRequireRejectsImportedNonPubResource(t *testing.T) {
	imported := Analyze(parseResourceTestContent(t, `
domain app

resource User {
    action read
}
`))
	content := parseResourceTestContent(t, `
domain user

import app

actor UserActor {
    via client {}
}

service UserService {
    for UserActor

    method getUser {
        require app.User:read
    }
}
`)

	defer func() {
		if recover() == nil {
			t.Fatalf("NewDomainWithImports() did not panic")
		}
	}()
	AnalyzeWithImports(content, []*Analysis{imported})
}
