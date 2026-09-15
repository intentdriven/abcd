package statusline

// The harness's status payload, reduced to the four values the row reads.
//
// The harness hands its status command a JSON object on stdin. This package
// never touches stdin — the front door reads the bytes and hands them here —
// and it deliberately keeps no model of the whole payload: it lifts four
// fields and ignores the rest, so a harness that grows a field, renames one it
// does not supply, or reorders the object changes nothing here.
//
// Every one of the four is OPTIONAL, and that is a property of the source
// rather than a defensive habit. The rate-limit object appears only for some
// plans and only after the first API response of a session, each of its
// windows is dropped independently once that window resets, and the context
// percentage is null before the first API call and again after a compaction.
// itd-200's commitment is that an absent field DROPS its element rather than
// showing a placeholder, so absence has to survive the parse as absence —
// which is why the three percentages are pointers and not zero-valued floats.
// A payload carrying 0% context is a different fact from one carrying none.

import (
	"encoding/json"
	"fmt"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// Payload is the parsed harness payload: what the four payload-sourced
// elements need, and nothing else. A nil pointer, or an empty Model, means the
// harness did not supply the field.
type Payload struct {
	// Model is the model's display name, falling back to its identifier when
	// the harness supplies only that. Sanitized.
	Model string `json:"model,omitempty"`
	// ContextPct is the percentage of the context window in use.
	ContextPct *float64 `json:"context_pct,omitempty"`
	// FiveHourPct and SevenDayPct are the percentages of the five-hour and
	// seven-day usage windows consumed. Either may read above 100.
	FiveHourPct *float64 `json:"five_hour_pct,omitempty"`
	SevenDayPct *float64 `json:"seven_day_pct,omitempty"`
	// Cwd is the directory the harness is standing in: the payload's `cwd`,
	// falling back to `workspace.current_dir`, empty when neither is supplied.
	// It exists so the status verb can resolve the checkout the HARNESS is in
	// rather than the one its own process happens to be in, and it is used for
	// filesystem resolution only — never rendered, never echoed, so it is not
	// sanitised and its tag keeps it out of every JSON envelope.
	Cwd string `json:"-"`
}

// harnessPayload is the wire shape, held privately so the exported Payload is
// abcd's own vocabulary rather than a harness's. Every level is a pointer:
// "the object is absent" and "the object is present with a null field" are
// both real states and reach the row identically, as an absence.
type harnessPayload struct {
	Model *struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow *struct {
		UsedPercentage *float64 `json:"used_percentage"`
	} `json:"context_window"`
	RateLimits *struct {
		FiveHour *struct {
			UsedPercentage *float64 `json:"used_percentage"`
		} `json:"five_hour"`
		SevenDay *struct {
			UsedPercentage *float64 `json:"used_percentage"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
	Cwd       *string `json:"cwd"`
	Workspace *struct {
		CurrentDir *string `json:"current_dir"`
	} `json:"workspace"`
}

// ParsePayload reads the harness's status payload.
//
// It refuses anything that is not a JSON object, the truncated stdin and the
// empty read included: a front door that mistook those for an empty payload
// would render a row that silently lost four elements, which is exactly the
// plausible-wrong-answer failure the drop rule exists to make visible. A
// caller that wants a row without payload elements passes a zero Payload.
func ParsePayload(data []byte) (Payload, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return Payload{}, fmt.Errorf("statusline: the status payload is not valid JSON: %w", err)
	}
	if probe == nil {
		return Payload{}, fmt.Errorf("statusline: the status payload is null, not an object")
	}
	var hp harnessPayload
	if err := json.Unmarshal(data, &hp); err != nil {
		return Payload{}, fmt.Errorf("statusline: the status payload is not the expected shape: %w", err)
	}

	var p Payload
	if hp.Model != nil {
		name := hp.Model.DisplayName
		if name == "" {
			name = hp.Model.ID
		}
		// Payload text that reaches a terminal goes through the repo's one
		// canonical mask; encoding/json passes DEL, the C1 range and the bidi
		// controls through untouched.
		p.Model = termsafe.Sanitize(name)
	}
	if hp.ContextWindow != nil {
		p.ContextPct = copyPct(hp.ContextWindow.UsedPercentage)
	}
	if hp.RateLimits != nil {
		if hp.RateLimits.FiveHour != nil {
			p.FiveHourPct = copyPct(hp.RateLimits.FiveHour.UsedPercentage)
		}
		if hp.RateLimits.SevenDay != nil {
			p.SevenDayPct = copyPct(hp.RateLimits.SevenDay.UsedPercentage)
		}
	}
	switch {
	case hp.Cwd != nil && *hp.Cwd != "":
		p.Cwd = *hp.Cwd
	case hp.Workspace != nil && hp.Workspace.CurrentDir != nil:
		p.Cwd = *hp.Workspace.CurrentDir
	}
	return p, nil
}

// copyPct returns an independent pointer, so the parsed Payload shares nothing
// with the intermediate wire struct.
func copyPct(p *float64) *float64 {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
