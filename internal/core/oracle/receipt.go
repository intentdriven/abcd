package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// The two places a Route leaves this package (spc-2609180535002478 step 4):
// the request block a delegating verb hands the host before the step runs, and
// the receipt its ingest returns after. Both are built from the Route Resolve
// returned, so no front door invents a route.

// Bounds on what a routing file or a --route may carry, enforced where the
// value is read (review-tier1 F6): everything that reaches a request block or
// a receipt has already passed them, so the receipt records settings as sent,
// whole, and never a truncation of them.
const (
	// MaxSettingBytes bounds one setting's value, as JSON.
	MaxSettingBytes = 256
	// MaxSettings bounds how many settings one row or one --route carries.
	MaxSettings = 16
	// MaxRouteBytes bounds one --route's text, which a receipt carries verbatim.
	MaxRouteBytes = 1024
	// MaxModelBytes bounds the model a payload reports, in a receipt.
	MaxModelBytes = 200
)

// RequestRouting is the request block's routing section for the one agent a
// step dispatches: the tier and fan-out bound the host is asked to honour, the
// layer that decided them, and the leg the step goes to (AC 1, AC 4).
type RequestRouting struct {
	Agent  string `json:"agent"`
	Tier   Tier   `json:"tier"`
	FanOut int    `json:"fan_out"`
	// Source is the deciding layer: flag, repo, machine, bundled, or none.
	Source string `json:"source"`
	// Origin names that layer's file, the flag text, "bundled" or "none".
	Origin string `json:"origin"`
	// Override is the --route text verbatim when one governed the step.
	Override string `json:"override,omitempty"`
	// Connection is the leg: a provider connection's name, or harness.
	Connection string `json:"connection"`
	// Fallback is why a tier that asked for a provider went to the harness.
	Fallback string `json:"fallback,omitempty"`
}

// Request returns the route's request-block section.
func (r Route) Request() RequestRouting {
	return RequestRouting{
		Agent: r.Agent, Tier: r.Row.Tier, FanOut: r.Row.FanOut,
		Source: r.Source.String(), Origin: r.Origin, Override: r.Override,
		Connection: r.ConnectionUsed, Fallback: r.Fallback,
	}
}

// ReceiptRoute is the receipt's route block (AC 4, 5, 7, 8): what was asked,
// which connection was tried and which used, why a step fell back, the
// override verbatim, the settings as sent, and the model the payload reported,
// side by side. Every field is always present, so a reader tells "none" from
// "not recorded".
type ReceiptRoute struct {
	Agent           string   `json:"agent"`
	TierAsked       Tier     `json:"tier_asked"`
	ConnectionTried string   `json:"connection_tried"`
	ConnectionUsed  string   `json:"connection_used"`
	FallbackReason  string   `json:"fallback_reason"`
	Override        string   `json:"override"`
	SettingsSent    Settings `json:"settings_sent"`
	// ModelReported is the payload's own model field, "" when it names none
	// (requiring it is itd-2609180517121254's).
	ModelReported string `json:"model_reported"`
}

// Receipt returns the route's receipt block, with the model the payload
// reported (ModelReported's answer).
func (r Route) Receipt(model string) ReceiptRoute {
	sent := r.SettingsSent
	if sent == nil {
		sent = Settings{}
	}
	return ReceiptRoute{
		Agent: r.Agent, TierAsked: r.Row.Tier,
		ConnectionTried: r.ConnectionTried, ConnectionUsed: r.ConnectionUsed,
		FallbackReason: r.Fallback, Override: r.Override,
		SettingsSent: sent, ModelReported: model,
	}
}

// RenderRequestSection renders the routing section a request document carries,
// as indented key: value lines under "routing:". Every value is sanitised: the
// document is read by the host, and a value a file or a flag supplied must not
// carry a terminal escape or a reordering rune into it.
func RenderRequestSection(rr RequestRouting) string {
	var b strings.Builder
	b.WriteString("routing:\n")
	line := func(k, v string) { fmt.Fprintf(&b, "  %s: %s\n", k, termsafe.Sanitize(v)) }
	line("agent", rr.Agent)
	line("tier", string(rr.Tier))
	line("fan_out", fmt.Sprint(rr.FanOut))
	line("source", rr.Source)
	line("origin", rr.Origin)
	if rr.Override != "" {
		line("override", rr.Override)
	}
	line("connection", rr.Connection)
	if rr.Fallback != "" {
		line("fallback", rr.Fallback)
	}
	return b.String()
}

// ModelReported returns the model a payload names: its top-level "model"
// string, else a reading's "instrument.model". A payload that names none, or
// is not a JSON object, reports "". The value is untrusted, so it is bounded
// to MaxModelBytes and its hidden runes are percent-encoded: recorded, never
// able to reorder or escape the receipt it lands in.
func ModelReported(payload []byte) string {
	var top map[string]json.RawMessage
	if json.Unmarshal(bytes.TrimSpace(payload), &top) != nil {
		return ""
	}
	m := stringField(top["model"])
	if m == "" {
		var inst map[string]json.RawMessage
		if json.Unmarshal(top["instrument"], &inst) == nil {
			m = stringField(inst["model"])
		}
	}
	m = termsafe.EncodeHiddenRunes(m)
	if len(m) > MaxModelBytes {
		cut := MaxModelBytes
		for cut > 0 && !isRuneStart(m[cut]) {
			cut--
		}
		m = m[:cut] + "..."
	}
	return m
}

func stringField(raw json.RawMessage) string {
	var s string
	if len(raw) == 0 || json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
