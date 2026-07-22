package analyzer

import (
	"slices"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

// MaxDiagnosticsPerDomain bounds validation work and prevents a badly broken
// source from flooding editor and command-line output.
const MaxDiagnosticsPerDomain = 50

type diagnosticReporter struct {
	errors []error
	seen   map[string]bool
}

func newDiagnosticReporter() *diagnosticReporter {
	return &diagnosticReporter{seen: map[string]bool{}}
}

func (r *diagnosticReporter) check(condition bool, message string, args ...any) bool {
	if condition {
		return true
	}
	r.report(checkutil.NewFailuref(message, args...))
	return false
}

func (r *diagnosticReporter) checkNot(condition bool, message string, args ...any) bool {
	return r.check(!condition, message, args...)
}

func (r *diagnosticReporter) reportf(message string, args ...any) {
	failure := checkutil.NewFailuref(message, args...)
	if strings.Contains(strings.ToLower(message), "duplicated") {
		failure.Code = "semantic.duplicate"
		positions := diagnosticArgumentPositions(args)
		if len(positions) > 1 {
			failure.Related = []checkutil.RelatedLocation{{Position: positions[len(positions)-1], Message: "first declaration"}}
		}
	}
	r.report(failure)
}

func diagnosticArgumentPositions(args []any) []model.Position {
	positions := []model.Position{}
	for _, argument := range args {
		switch position := argument.(type) {
		case model.Position:
			positions = append(positions, position)
		case lexer.Position:
			positions = append(positions, model.Position{File: position.Filename, Line: position.Line, Column: position.Column})
		}
	}
	return positions
}

func (r *diagnosticReporter) report(err error) {
	if err == nil || len(r.errors) >= MaxDiagnosticsPerDomain {
		return
	}
	position, _ := checkutil.Position(err)
	key := position.String() + "\x00" + err.Error()
	if r.seen[key] {
		return
	}
	r.seen[key] = true
	r.errors = append(r.errors, err)
}

func (r *diagnosticReporter) full() bool {
	return len(r.errors) >= MaxDiagnosticsPerDomain
}

func (r *diagnosticReporter) result() []error {
	result := append([]error{}, r.errors...)
	slices.SortFunc(result, func(left, right error) int {
		leftPosition, _ := checkutil.Position(left)
		rightPosition, _ := checkutil.Position(right)
		if compared := strings.Compare(leftPosition.File, rightPosition.File); compared != 0 {
			return compared
		}
		if leftPosition.Line != rightPosition.Line {
			return leftPosition.Line - rightPosition.Line
		}
		if leftPosition.Column != rightPosition.Column {
			return leftPosition.Column - rightPosition.Column
		}
		return strings.Compare(left.Error(), right.Error())
	})
	return result
}
