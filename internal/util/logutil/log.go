package logutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Level string

const (
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

const (
	FormatText  = "text"
	FormatJSONL = "jsonl"
)

type Entry struct {
	Level   Level
	Message string
}

var entries []Entry

func Clear() {
	entries = nil
}

type _JSONLEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

func Info(format string, args ...any) Entry {
	return appendEntry(LevelInfo, format, args...)
}

func Warn(format string, args ...any) Entry {
	return appendEntry(LevelWarn, format, args...)
}

func Error(format string, args ...any) Entry {
	return appendEntry(LevelError, format, args...)
}

func Entries() []Entry {
	return append([]Entry{}, entries...)
}

func Format(entry Entry, format string) string {
	message := strings.TrimSpace(entry.Message)
	if format == FormatJSONL {
		return jsonl(entry.Level, message)
	}
	switch entry.Level {
	case LevelInfo:
		return "[I] " + stripPrefix(message, "[I] ")
	case LevelWarn:
		return "[W] " + stripPrefix(message, "[W] ")
	case LevelError:
		return "Error: " + message
	default:
		return message
	}
}

func appendEntry(level Level, format string, args ...any) Entry {
	entry := Entry{
		Level:   level,
		Message: fmt.Sprintf(format, args...),
	}
	entries = append(entries, entry)
	return entry
}

func jsonl(level Level, message string) string {
	data, err := json.Marshal(_JSONLEntry{Level: string(level), Message: normalizeMessage(level, message)})
	if err != nil {
		panic(fmt.Sprintf("marshal log entry: %v", err))
	}
	return string(data) + "\n"
}

func normalizeMessage(level Level, message string) string {
	message = strings.TrimSpace(message)
	switch level {
	case LevelInfo:
		return stripPrefix(message, "[I] ")
	case LevelWarn:
		return stripPrefix(message, "[W] ")
	default:
		return message
	}
}

func stripPrefix(message string, prefix string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(message), prefix))
}
