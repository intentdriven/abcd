package loop

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Refusal is the one shape every refusal of the loop takes (criterion 13): the
// stage it happened at, the reason and the remedy, in text and in --json. A
// refusal writes nothing: the state file is as it was before the call, except
// for the time a landing began waiting for its full check (ruling DR6d-2),
// which the call that finds the wait writes once, whatever the call answers.
type Refusal struct {
	// Stage is where the loop refused: "check" before a run starts, "state" for
	// a run that cannot be read, "pause" for the window clock, or the lane stage
	// ("worktree", "implement", …) and "receipt" inside a run.
	Stage string `json:"stage"`
	// Check names the pre-start check that failed, when Stage is "check".
	Check string `json:"check,omitempty"`
	// Lane names the lane, inside a run.
	Lane   string `json:"lane,omitempty"`
	Reason string `json:"reason"`
	Remedy string `json:"remedy"`
	// Contention marks a refusal that is somebody else's hold rather than the
	// caller's fault — a peer holding the record, a run locked by another
	// invocation, a pause: back off and take other work. The CLI maps it to
	// exit 3, as `abcd implement` does.
	Contention bool `json:"contention,omitempty"`
	// Checks is every pre-start check's row, when Stage is "check", so a caller
	// sees the whole picture rather than the first failure.
	Checks []CheckRow `json:"checks,omitempty"`
	// Excluded is every planned intent a pick excluded and the check that
	// excluded it, when a pick found no candidate.
	Excluded []Excluded `json:"excluded,omitempty"`
	// checkWait is when a landing began waiting for its full check (ruling
	// DR6d-2): the call that finds the wait writes it on the lane once.
	checkWait time.Time
}

// Error renders the refusal as one line: stage, reason, remedy.
func (r *Refusal) Error() string {
	var b strings.Builder
	b.WriteString("refused at ")
	b.WriteString(r.Stage)
	if r.Check != "" {
		b.WriteString(" (" + r.Check + ")")
	}
	if r.Lane != "" {
		b.WriteString(" for " + r.Lane)
	}
	b.WriteString(": " + r.Reason)
	if r.Remedy != "" {
		b.WriteString("; remedy: " + r.Remedy)
	}
	return b.String()
}

// AsRefusal returns err's Refusal, if it carries one.
func AsRefusal(err error) (*Refusal, bool) {
	var r *Refusal
	if errors.As(err, &r) {
		return r, true
	}
	return nil, false
}

// refuse builds a Refusal.
func refuse(stage, check, lane, reason, remedy string) *Refusal {
	return &Refusal{Stage: stage, Check: check, Lane: lane, Reason: reason, Remedy: remedy}
}

// contend builds a contention Refusal.
func contend(stage, check, lane, reason, remedy string) *Refusal {
	r := refuse(stage, check, lane, reason, remedy)
	r.Contention = true
	return r
}

// refusef is refuse with a formatted reason.
func refusef(stage, lane, remedy, format string, a ...any) *Refusal {
	return refuse(stage, "", lane, fmt.Sprintf(format, a...), remedy)
}
