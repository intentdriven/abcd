package ask

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/question"
)

// Safe returns a copy of a with every part sanitised for the screen
// (spc-2610030911534855, "Sanitising before drawing"): question.Safe, the one
// copy the drawing and the AI-written interviews' check both use, so a
// question is measured and drawn as the same text.
func Safe(a question.Ask) question.Ask { return question.Safe(a) }

// Refusal is a question the structural check refuses: it is never drawn.
type Refusal struct {
	Findings []question.Finding
}

func (r *Refusal) Error() string {
	parts := make([]string, len(r.Findings))
	for i, f := range r.Findings {
		parts[i] = f.String()
	}
	return "the question cannot be drawn: " + strings.Join(parts, "; ")
}

// Prepare holds a to the structural check (question.Check) and returns it
// sanitised (Safe): the one door a question passes before it is drawn. Until
// the companion bundle's limits are run on this path, a question drawn in a
// plain Terminal is held to the structural check alone (spc-2610030911534855,
// Scope "Out"). A refused question is returned as a *Refusal naming every
// finding.
func Prepare(a question.Ask) (question.Ask, error) {
	if err := check(a); err != nil {
		return question.Ask{}, err
	}
	return Safe(a), nil
}

// check is Prepare's structural check alone, for a caller that sanitises as
// it draws (the answer loop keeps the question as given for its answers).
func check(a question.Ask) error {
	if fs := question.Check(a); len(fs) > 0 {
		return &Refusal{Findings: fs}
	}
	return nil
}
