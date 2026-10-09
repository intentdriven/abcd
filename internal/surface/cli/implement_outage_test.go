package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// outageFixture stands up a run in a committed repository under a temporary
// HOME, with the verb's clock and network probe swapped for the test's: no real
// waiting, no network. It returns the clock and the switch the probe reads.
func outageFixture(t *testing.T) (now *time.Time, netUp *bool) {
	t.Helper()
	implementRepo(t)
	clock := time.Date(2026, 10, 9, 23, 0, 0, 0, time.UTC)
	up := false
	prevNow, prevProbe := outageNow, outageNetworkProbe
	outageNow = func() time.Time { return clock }
	outageNetworkProbe = func(string) func() (bool, string) {
		return func() (bool, string) {
			if up {
				return true, "origin answered"
			}
			return false, "Could not resolve host: github.com"
		}
	}
	t.Cleanup(func() { outageNow, outageNetworkProbe = prevNow, prevProbe })
	mustImplement(t, "implement", "join", "--session", "alpha", "--role", "first", "--json")
	return &clock, &up
}

// outageState reads the bare verb's --json.
func outageState(t *testing.T) implementOutageOutput {
	t.Helper()
	var st implementOutageOutput
	if err := json.Unmarshal([]byte(mustImplement(t, "implement", "outage", "--json")), &st); err != nil {
		t.Fatal(err)
	}
	return st
}

// probeJSON runs `outage probe --json` and returns its exit code and output.
func probeJSON(t *testing.T, extra ...string) (int, map[string]any) {
	t.Helper()
	code, out, errOut := implementCLI(t, append([]string{"implement", "outage", "probe", "--session", "alpha", "--json"}, extra...)...)
	var v map[string]any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("probe --json is not JSON (exit %d): %q %q", code, out, errOut)
	}
	return code, v
}

// TestImplementOutageVerb: the front door onto the shared outage, through the
// CLI — the read-only status, a report, the probe's three exits (0 go ahead,
// 3 wait with next_probe_at, 2 the run gave up), the model service proven only
// by --model, the hand clear, the acknowledgement, and the report's outages.
func TestImplementOutageVerb(t *testing.T) {
	now, netUp := outageFixture(t)

	if out := mustImplement(t, "implement", "outage"); !strings.Contains(out, "no outage") {
		t.Fatalf("bare with no outage:\n%s", out)
	}
	if st := outageState(t); st.Outage != nil {
		t.Fatalf("bare --json with no outage = %+v", st.Outage)
	}
	if code, v := probeJSON(t); code != 0 || v["probed"] != false {
		t.Fatalf("a probe with no outage = %d %v; want exit 0, nothing probed", code, v)
	}
	if code, _, _ := implementCLI(t, "implement", "outage", "record", "--service", "network", "--kind", "tool", "--lane", "l1", "--what", "x"); code != 2 {
		t.Fatalf("record without --session exited %d; want 2", code)
	}
	if msg := refusalEnvelope(t, 2, "implement", "outage", "record", "--session", "alpha", "--service", "network",
		"--kind", "wifi", "--lane", "l1", "--what", "x", "--json"); !strings.Contains(msg, "kind") {
		t.Fatalf("an unknown kind = %q", msg)
	}
	mustImplement(t, "implement", "outage", "record", "--session", "alpha", "--service", "network", "--kind", "tool",
		"--lane", "lane-1", "--what", "git push", "--json")
	out := mustImplement(t, "implement", "outage")
	for _, want := range []string{"outage open since 2026-10-09T23:00:00Z", "down:", "network", "next probe: 2026-10-09T23:01:00Z", "git push"} {
		if !strings.Contains(out, want) {
			t.Errorf("bare render lacks %q:\n%s", want, out)
		}
	}

	// Not due: exit 3, and the JSON says when to come back.
	code, v := probeJSON(t)
	if code != 3 || v["waiting"] != true || v["next_probe_at"] != "2026-10-09T23:01:00Z" {
		t.Fatalf("a probe before it is due = %d %v; want exit 3 with next_probe_at", code, v)
	}
	// Due and still down: exit 3, the next probe five minutes on.
	*now = now.Add(time.Minute)
	code, v = probeJSON(t)
	if code != 3 || v["probed"] != true || outageState(t).Outage.NextProbeAt.Format(time.RFC3339) != "2026-10-09T23:06:00Z" {
		t.Fatalf("a failed probe = %d %v", code, v)
	}
	// The model service goes down too; the network comes back, but without the
	// canary's verdict the outage holds.
	mustImplement(t, "implement", "outage", "record", "--session", "alpha", "--service", "model", "--kind", "agent",
		"--lane", "lane-2", "--what", "API Error: 529 Overloaded")
	*netUp = true
	*now = now.Add(5 * time.Minute)
	if code, _ = probeJSON(t); code != 3 {
		t.Fatalf("network back, model unproven: exit %d; want 3", code)
	}
	if st := outageState(t); len(st.Outage.Down) != 1 || st.Outage.Down[0] != "model" {
		t.Fatalf("down after the network came back = %v", st.Outage.Down)
	}
	if msg := refusalEnvelope(t, 2, "implement", "outage", "probe", "--session", "alpha", "--model", "maybe", "--json"); !strings.Contains(msg, "--model") {
		t.Fatalf("--model maybe = %q", msg)
	}
	*now = now.Add(10 * time.Minute)
	code, v = probeJSON(t, "--model", "ok")
	if code != 0 || v["ended"] != true || v["minutes"] != 16.0 {
		t.Fatalf("the canary back = %d %v; want exit 0 and the outage ended after 16 minutes", code, v)
	}
	if out := mustImplement(t, "implement", "outage"); !strings.Contains(out, "no outage") {
		t.Fatalf("bare after the end:\n%s", out)
	}

	// A hand clear needs its reason.
	mustImplement(t, "implement", "outage", "record", "--session", "alpha", "--service", "network", "--kind", "host",
		"--lane", "lane-3", "--what", "model call")
	refusalEnvelope(t, 2, "implement", "outage", "clear", "--session", "alpha", "--json")
	if out := mustImplement(t, "implement", "outage", "clear", "--session", "alpha", "--reason", "router restarted"); !strings.Contains(out, "cleared") {
		t.Fatalf("clear:\n%s", out)
	}

	// The give-up: exit 2, the notification pending until acknowledged, once.
	*netUp = false
	mustImplement(t, "implement", "outage", "record", "--session", "alpha", "--service", "network", "--kind", "tool",
		"--lane", "lane-1", "--what", "gh pr create")
	for i := 0; ; i++ {
		if i > 20 {
			t.Fatal("the probe never gave up")
		}
		*now = outageState(t).Outage.NextProbeAt
		if code, v = probeJSON(t); code == 2 {
			break
		}
	}
	if v["gave_up"] != true {
		t.Fatalf("the give-up probe = %v", v)
	}
	if out := mustImplement(t, "implement", "outage"); !strings.Contains(out, "gave up") || !strings.Contains(out, "notification pending") {
		t.Fatalf("bare after the give-up:\n%s", out)
	}
	if msg := refusalEnvelope(t, 2, "implement", "outage", "probe", "--session", "alpha", "--json"); !strings.Contains(msg, "gave up") {
		t.Fatalf("a probe after the give-up = %q", msg)
	}
	mustImplement(t, "implement", "outage", "ack", "--session", "alpha")
	refusalEnvelope(t, 2, "implement", "outage", "ack", "--session", "alpha", "--json")

	out = mustImplement(t, "implement", "report")
	for _, want := range []string{"outage: 2026-10-09T23:00:00Z", "16.0 min", "ended", "cleared", "gave up", "retried: git push; API Error: 529 Overloaded"} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
	if out := mustImplement(t, "implement", "report", "--json"); !strings.Contains(out, `"outages": [`) || !strings.Contains(out, `"outcome": "gave_up"`) {
		t.Errorf("report --json lacks the outages:\n%s", out)
	}
}
