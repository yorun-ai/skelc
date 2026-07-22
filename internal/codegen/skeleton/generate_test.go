package skeleton

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	skelparser "go.yorun.ai/skelc/internal/parser"
	"go.yorun.ai/skelc/model"
)

func TestGenWritesDomainDescriptionAndOnlyNeededImports(t *testing.T) {
	_, userDir := parseDomainForTest(t, "user/domain.skel", "domain user\n", "user/types.skel", `domain user

pub data UserSummary {
    id: string
}
`, nil)
	bookerDomain, _ := parseDomainForTest(t, "booker/domain.skel", `@desc("""
Library management example
Skel definitions demonstrating catalogs, loans, tasks, and events
""")
domain booker
`, "booker/types.skel", `domain booker

import user as account

pub data Loan {
    borrower: account.UserSummary
}
`, map[string]string{"user": userDir})
	outputDir := t.TempDir()

	Generate(bookerDomain, Option{Out: outputDir, PubOnly: true})

	domainContent := readGeneratedFileForTest(t, filepath.Join(outputDir, "domain.skel"))
	if strings.Contains(domainContent, "import user") {
		t.Fatalf("domain.skel should not include dependency imports:\n%s", domainContent)
	}
	if !strings.Contains(domainContent, `@desc("""
Library management example
Skel definitions demonstrating catalogs, loans, tasks, and events
""")
domain booker
`) {
		t.Fatalf("expected multiline domain description to stay triple quoted:\n%s", domainContent)
	}

	typesContent := readGeneratedFileForTest(t, filepath.Join(outputDir, "types.skel"))
	if !strings.Contains(typesContent, "import user as account") {
		t.Fatalf("types.skel should include used dependency import:\n%s", typesContent)
	}
}

func TestGenKeepsDescriptionIndentation(t *testing.T) {
	domain, _ := parseDomainForTest(t, "demo/domain.skel", "domain demo\n", "demo/all.skel", `domain demo

@desc("""
User profile
Multiline description
""")
pub data User {
    @desc("""
    User ID
    Multiline field description
    """)
    id: string
}

pub data Page<TItem> {
    items: map<string, list<TItem>>
    nextToken: string?
}

pub config CacheConfig eternal {
    enabled: bool
}

pub actor ClientActor {
    via client {}
    auth {
        info {
            userId: string
        }
        credential {
            subject: string
        }
    }
}

pub service UserService {
    for ClientActor via client

    @desc("Health check")
    method ping {
    }

    @desc("""
    Query a user
    Multiline method description
    """)
    method getUser {
        @desc("""
        Input parameters
        Multiline input description
        """)
        input {
            @desc("""
            User ID
            Multiline parameter description
            """)
            id: string
        }
        @desc("The returned user entity")
        @example({
id:10001,
username:"zhangsan",
displayName:"Alice",
email:"a@b.com",
})
        output User
    }

    method listUsers {
        output list<User>
    }
}
`, nil)
	outputDir := t.TempDir()

	Generate(domain, Option{Out: outputDir, PubOnly: true})
	assertGeneratedSkelParses(t, outputDir)

	typesContent := readGeneratedFileForTest(t, filepath.Join(outputDir, "types.skel"))
	assertNoExtraTopLevelBlankLines(t, typesContent)
	if !strings.Contains(typesContent, `pub data User {
    @desc("""
    User ID
    Multiline field description
    """)
    id: string
}`) {
		t.Fatalf("expected member desc indentation, got:\n%s", typesContent)
	}
	if strings.Contains(typesContent, "ClientActorCredential") || strings.Contains(typesContent, "ClientActorInfo") {
		t.Fatalf("actor implicit data should stay in actor.skel, got:\n%s", typesContent)
	}
	if !strings.Contains(typesContent, "pub data Page<TItem> {\n    items: map<string, list<TItem>>\n    nextToken: string?\n}") {
		t.Fatalf("expected template-rendered generic types, got:\n%s", typesContent)
	}
	if !strings.Contains(typesContent, "pub config CacheConfig eternal {") {
		t.Fatalf("expected template-rendered config lifecycle, got:\n%s", typesContent)
	}

	actorContent := readGeneratedFileForTest(t, filepath.Join(outputDir, "actor.skel"))
	if !strings.Contains(actorContent, "    auth {\n        credential {\n            subject: string\n        }\n        info {\n            userId: string\n        }\n    }") {
		t.Fatalf("expected actor credential and info to render in stable order, got:\n%s", actorContent)
	}

	serviceContent := readGeneratedFileForTest(t, filepath.Join(outputDir, "service.skel"))
	assertNoExtraTopLevelBlankLines(t, serviceContent)
	if !strings.Contains(serviceContent, "    for ClientActor via client\n") {
		t.Fatalf("expected service for via to render, got:\n%s", serviceContent)
	}
	if !strings.Contains(serviceContent, "    for ClientActor via client\n\n    @desc(\"Health check\")\n    method ping {}\n") {
		t.Fatalf("expected blank line after service for block, got:\n%s", serviceContent)
	}
	if !strings.Contains(serviceContent, `    @desc("""
    Query a user
    Multiline method description
    """)
    method getUser {
        @desc("""
        Input parameters
        Multiline input description
        """)
        input {
            @desc("""
            User ID
            Multiline parameter description
            """)
            id: string
        }
        @desc("The returned user entity")
        @example({
            id:10001,
            username:"zhangsan",
            displayName:"Alice",
            email:"a@b.com",
        })
        output User
    }`) {
		t.Fatalf("expected service desc indentation, got:\n%s", serviceContent)
	}
	if !strings.Contains(serviceContent, "    }\n\n    method listUsers {\n") {
		t.Fatalf("expected blank line between service methods, got:\n%s", serviceContent)
	}
	if strings.Contains(serviceContent, "pub service UserService {\n\n") {
		t.Fatalf("did not expect blank line before first service for, got:\n%s", serviceContent)
	}
	if !strings.Contains(serviceContent, "    method ping {}\n") {
		t.Fatalf("expected empty method to render on one line, got:\n%s", serviceContent)
	}
}

func TestGenResourceHidesPermissionCodeArgument(t *testing.T) {
	domain, _ := parseDomainForTest(t, "demo/domain.skel", "domain demo\n", "demo/types.skel", `domain demo

@desc("user")
pub resource User {
    check byExists(userId: int)

    @desc("read user")
    action read
    @desc("update user")
    action update {
        check bySelf(userId: int)
    }
    @desc("manage user")
    action manage {
        check byNotSelf(userId: int)
    }
}
`, nil)
	outputDir := t.TempDir()

	Generate(domain, Option{Out: outputDir, PubOnly: true})

	typesContent := readGeneratedFileForTest(t, filepath.Join(outputDir, "types.skel"))
	if strings.Contains(typesContent, "PermissionCode") || strings.Contains(typesContent, "code:") {
		t.Fatalf("resource check generated skel should hide internal code argument, got:\n%s", typesContent)
	}
	if !strings.Contains(typesContent, "    check byExists(userId: int)\n\n    @desc(\"read user\")") {
		t.Fatalf("expected blank line between resource checks and actions, got:\n%s", typesContent)
	}
	if !strings.Contains(typesContent, "        check bySelf(userId: int)") {
		t.Fatalf("expected action check arguments without code, got:\n%s", typesContent)
	}
	if !strings.Contains(typesContent, "    }\n\n    @desc(\"manage user\")") {
		t.Fatalf("expected blank line between resource actions, got:\n%s", typesContent)
	}
	if strings.Contains(typesContent, "        check bySelf(userId: int)\n\n    }") {
		t.Fatalf("did not expect blank line before resource action closing brace, got:\n%s", typesContent)
	}
}

func TestGenDoesNotRenderBlankLineAfterServiceAudienceWithoutFollowingContent(t *testing.T) {
	domain := model.NewDomainFromSpec(model.DomainSpec{
		Name: "demo",
		Actors: []*model.Actor{
			{Name: "ClientActor", Pub: true},
		},
		Services: []*model.Service{
			{
				Name:      "EmptyService",
				Pub:       true,
				Audiences: []*model.ActorAudience{{Actor: "ClientActor"}},
			},
		},
	})
	outputDir := t.TempDir()

	Generate(domain, Option{Out: outputDir, PubOnly: true})

	serviceContent := readGeneratedFileForTest(t, filepath.Join(outputDir, "service.skel"))
	if !strings.Contains(serviceContent, "pub service EmptyService {\n    for ClientActor\n}") {
		t.Fatalf("did not expect blank line after for without following content, got:\n%s", serviceContent)
	}
}

func parseDomainForTest(t *testing.T, domainPath string, domainContent string, inputPath string, inputContent string, imports map[string]string) (*model.Domain, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(domainPath)), []byte(domainContent), 0o644); err != nil {
		t.Fatalf("write domain fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.Base(inputPath)), []byte(inputContent), 0o644); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	parsed, err := skelparser.Parse(skelparser.Option{SkelIn: dir, SkelImports: imports})
	if err != nil {
		t.Fatalf("parse test domain: %v", err)
	}
	return parsed.Domain, dir
}

func readGeneratedFileForTest(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated file %s: %v", path, err)
	}
	return string(content)
}

func assertGeneratedSkelParses(t *testing.T, outputDir string) {
	t.Helper()
	if _, err := skelparser.Parse(skelparser.Option{SkelIn: outputDir}); err != nil {
		t.Fatalf("parse generated skel: %v", err)
	}
}

func assertNoExtraTopLevelBlankLines(t *testing.T, content string) {
	t.Helper()
	for _, unexpected := range []string{
		"\n\n\n@desc",
		"\n\n\npub ",
	} {
		if strings.Contains(content, unexpected) {
			t.Fatalf("unexpected extra blank line in generated skel:\n%s", content)
		}
	}
}
