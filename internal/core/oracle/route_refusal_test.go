package oracle

import (
	"strings"
	"testing"
)

// TestARouteRefusalNamesWhatWasTypedByLength: every refusal of a --route value
// names it by its position, and a part the machine does not know (an agent off
// the roster, a connection, a setting's name or value, a model) by its length,
// never by its text: a verb's --json refusal document carries the refusal
// where the error boundary's pass does not read it, and the boundary withholds
// whole arguments only, never a part of one.
func TestARouteRefusalNamesWhatWasTypedByLength(t *testing.T) {
	const tok = "zqtokenshapedvalue7x"
	cases := []struct {
		name   string
		runner bool
		route  string
		want   string
	}{
		{"agent off the roster", false, tok + "=frontier", "number 1"},
		{"tier outside the vocabulary", false, "scribe=" + tok, "host-decides"},
		{"tier outside the vocabulary, with a connection", false, "scribe=" + tok + "@desk", "host-decides"},
		{"connection not configured", false, "scribe=frontier@" + tok, "connection"},
		{"setting not k=v", false, "scribe=frontier?" + tok, "not k=v"},
		{"setting name not a parameter", false, "scribe=frontier?" + strings.ToUpper(tok) + "=1", "parameter name"},
		// A --route's setting value is always read as a string (settingValue),
		// so the scalar check is a routing file's alone.
		{"setting value too long", false, "scribe=frontier?seed=" + tok + strings.Repeat("v", MaxSettingBytes), "seed is"},
		{"long setting name", false, "scribe=frontier?" + tok + "=" + strings.Repeat("v", MaxSettingBytes+1), "byte name"},
		{"setting key twice", false, "scribe=frontier?" + tok + "=1," + tok + "=2", "more than once"},
		{"runner route: no equals sign", true, tok, "number 1"},
		{"runner route: tier", true, tok + "=frontier", "names a tier"},
		{"runner route: no runner rung", true, tok + "=opencode", "names no runner"},
		{"runner route: model shape", true, "implementer=runner:opencode@" + tok, "<provider>/<model>"},
		{"runner route: role not dispatched", true, tok + "=runner:opencode", "dispatches"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var err error
			if c.runner {
				_, err = ParseRunnerRoutes([]string{c.route}, []string{"implementer"})
			} else {
				_, err = ParseRoutes([]string{c.route}, []string{"scribe"}, NoConnections{})
			}
			if err == nil {
				t.Fatal("the route was admitted")
			}
			msg := err.Error()
			if strings.Contains(strings.ToLower(msg), tok) {
				t.Fatalf("the refusal repeats what was typed: %s", msg)
			}
			if !strings.Contains(msg, "--route number 1") || !strings.Contains(msg, c.want) {
				t.Fatalf("the refusal does not name the route by position and %q: %s", c.want, msg)
			}
		})
	}
}

// TestARouteRefusalStillNamesARosterAgent: an agent of the roster that the
// invocation does not dispatch is named, since it is a name the roster knows,
// not only what the caller typed.
func TestARouteRefusalStillNamesARosterAgent(t *testing.T) {
	_, err := ParseRoutes([]string{"scribe=frontier"}, []string{"intent-auditor"}, NoConnections{})
	if err == nil || !strings.Contains(err.Error(), `not "scribe"`) {
		t.Fatalf("the refusal does not name the roster agent: %v", err)
	}
}
