package logutil

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestErrorBuildsErrorEntry(t *testing.T) {
	entry := Error("failed to %s the contract", "compile")
	if entry.Level != LevelError {
		t.Fatalf("unexpected level: %q", entry.Level)
	}
	if entry.Message != "failed to compile the contract" {
		t.Fatalf("unexpected message: %q", entry.Message)
	}
}

func TestFormatRendersTextLevels(t *testing.T) {
	for _, test := range []struct {
		name  string
		entry Entry
		want  string
	}{
		{name: "info", entry: Entry{Level: LevelInfo, Message: "  [I] ready  "}, want: "[I] ready"},
		{name: "info without prefix", entry: Entry{Level: LevelInfo, Message: "ready"}, want: "[I] ready"},
		{name: "warn", entry: Entry{Level: LevelWarn, Message: "[W] slow"}, want: "[W] slow"},
		{name: "error keeps text", entry: Entry{Level: LevelError, Message: "  boom  "}, want: "Error: boom"},
		{name: "unknown level", entry: Entry{Level: "trace", Message: "  detail  "}, want: "detail"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := Format(test.entry, FormatText); got != test.want {
				t.Fatalf("Format() = %q; want %q", got, test.want)
			}
		})
	}
}

func TestFormatRendersJSONL(t *testing.T) {
	got := Format(Entry{Level: LevelInfo, Message: "  [I] ready  "}, FormatJSONL)
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("JSONL entries must end with a newline: %q", got)
	}

	var entry _JSONLEntry
	if err := json.Unmarshal([]byte(got), &entry); err != nil {
		t.Fatalf("decode JSONL entry: %v", err)
	}
	if entry.Level != "info" || entry.Message != "ready" {
		t.Fatalf("unexpected JSONL entry: %+v", entry)
	}
}

func TestFormatKeepsErrorTextInJSONL(t *testing.T) {
	got := Format(Entry{Level: LevelError, Message: "  Error: boom  "}, FormatJSONL)

	var entry _JSONLEntry
	if err := json.Unmarshal([]byte(got), &entry); err != nil {
		t.Fatalf("decode JSONL entry: %v", err)
	}
	if entry.Level != "error" || entry.Message != "Error: boom" {
		t.Fatalf("unexpected JSONL entry: %+v", entry)
	}
}
