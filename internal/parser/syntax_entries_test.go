package parser

import (
	"go.yorun.ai/skelc/model"
	"testing"
)

func TestParseServiceWithoutActor(t *testing.T) {
	domain := parseDomain(t, map[string]string{
		"domain.skel": "@desc(\"User domain\")\ndomain demo.user\n",
		"service.skel": `
service UserService {
    method ping {
        output string
    }
}
`,
	})

	if len(domain.Services()) != 1 {
		t.Fatalf("unexpected service count: %d", len(domain.Services()))
	}
	service := domain.Services()[0]
	if service.Name != "UserService" {
		t.Fatalf("unexpected service name: %s", service.Name)
	}
	if len(service.Audiences) != 0 {
		t.Fatalf("expected empty audiences, got %+v", service.Audiences)
	}
}

func TestParseEvent(t *testing.T) {
	domain := parseDomain(t, map[string]string{
		"domain.skel": "@desc(\"User domain\")\ndomain demo.user\n",
		"event.skel": `
actor PartnerActor { via client {} }
actor OpenAPIActor { via openapi {} }

@desc("User created event")
event UserCreatedEvent {
    payload {
        @desc("User ID")
        userId: int

        @desc("Created at")
        createdAt: localdatetime
    }
}
`,
	})

	if len(domain.Events()) != 1 {
		t.Fatalf("unexpected event count: %d", len(domain.Events()))
	}
	event := domain.Events()[0]
	if event.Name != "UserCreatedEvent" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if event.Kind != model.DataKindEvent {
		t.Fatalf("unexpected event kind: %v", event.Kind)
	}
	if len(event.Members) != 2 {
		t.Fatalf("unexpected event members: %+v", event.Members)
	}
}

func TestParseTask(t *testing.T) {
	domain := parseDomain(t, map[string]string{
		"domain.skel": "@desc(\"User domain\")\ndomain demo.user\n",
		"task.skel": `
@desc("Rebuild the user index")
task RebuildUserIndexTask {
	@desc("Scheduled trigger")
	trigger atTime {
		input {
			startAt: localdatetime
		}
	}

	@desc("Trigger by user group")
	trigger forGroup {
		input {
			groupId: int
		}
	}
}
`,
	})

	if len(domain.Tasks()) != 1 {
		t.Fatalf("unexpected task count: %d", len(domain.Tasks()))
	}
	task := domain.Tasks()[0]
	if task.Name != "RebuildUserIndexTask" || task.SkelName != "demo.user.RebuildUserIndexTask" {
		t.Fatalf("unexpected task: %+v", task)
	}
	if len(task.Triggers) != 2 {
		t.Fatalf("unexpected task trigger count: %d", len(task.Triggers))
	}
	if task.Triggers[0].Name != "atTime" || task.Triggers[1].Name != "forGroup" {
		t.Fatalf("unexpected task triggers: %+v", task.Triggers)
	}
}
