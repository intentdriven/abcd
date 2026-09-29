package cli

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// The per-event timeouts in hooks/hooks.json are a ruling, not a default to
// drift from (iss-323; the product thinker's ruling of 2026-09-29): an entry
// that runs hooks/bootstrap.sh as a salvage declares 120 seconds, because
// bootstrap's download worst case outlasts the host's default and a killed
// salvage stamps .bootstrap.attempt and suppresses the retry for ten minutes.
// SessionStart, the primary provisioner, declares 240. Every other entry
// declares none and takes the host's default. The wait must also never be
// silent: every salvage entry names a statusMessage, the host's spinner text
// shown while a hook runs, because the salvage command itself sends
// bootstrap's output to /dev/null.

// wantHookTimeouts pins every event the manifest declares and the timeout its
// one handler carries; 0 means the field is absent. An event added, removed or
// re-timed without updating this table fails the test.
var wantHookTimeouts = map[string]int{
	"SessionStart":     240,
	"UserPromptSubmit": 120,
	"PreToolUse":       120,
	"PreCompact":       120,
	"SessionEnd":       0,
	"SubagentStop":     0,
}

// salvageTimeout is the budget every salvage entry declares.
const salvageTimeout = 120

// decodedHooksManifest reads the committed manifest through the same decoded
// shape the SessionStart and self-provision tests use.
func decodedHooksManifest(t *testing.T) sessionStartHooks {
	t.Helper()
	data, err := os.ReadFile(hooksManifest(t))
	if err != nil {
		t.Fatalf("reading the committed hooks manifest: %v", err)
	}
	var doc sessionStartHooks
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("hooks/hooks.json does not parse: %v", err)
	}
	return doc
}

func TestEveryHookEventPinsItsTimeout(t *testing.T) {
	doc := decodedHooksManifest(t)
	var got, want []string
	for event := range doc.Hooks {
		got = append(got, event)
	}
	for event := range wantHookTimeouts {
		want = append(want, event)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("hooks/hooks.json declares events %v; the timeout table pins %v — pin the new event's timeout here", got, want)
	}
	for event, wantTimeout := range wantHookTimeouts {
		entries := doc.Hooks[event]
		if len(entries) != 1 || len(entries[0].Hooks) != 1 {
			t.Fatalf("%s must declare exactly one entry group holding one command", event)
		}
		h := entries[0].Hooks[0]
		switch {
		case wantTimeout == 0 && h.Timeout != nil:
			t.Errorf("%s declares timeout %d; it is pinned to the host's default (no timeout field)", event, *h.Timeout)
		case wantTimeout != 0 && h.Timeout == nil:
			t.Errorf("%s declares no timeout; it is pinned to %d seconds", event, wantTimeout)
		case wantTimeout != 0 && *h.Timeout != wantTimeout:
			t.Errorf("%s declares timeout %d; it is pinned to %d seconds", event, *h.Timeout, wantTimeout)
		}
	}
}

// TestEverySalvageEntryShowsItsWait derives the salvage entries from the
// manifest rather than from a list: every entry other than SessionStart whose
// command runs hooks/bootstrap.sh gets the 120-second budget and a spinner
// message, so a new salvage entry cannot arrive silent or on the default.
func TestEverySalvageEntryShowsItsWait(t *testing.T) {
	doc := decodedHooksManifest(t)
	var salvage []string
	for event, entries := range doc.Hooks {
		if event == "SessionStart" {
			continue
		}
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				if !strings.Contains(h.Command, "hooks/bootstrap.sh") {
					continue
				}
				salvage = append(salvage, event)
				if h.Timeout == nil || *h.Timeout != salvageTimeout {
					t.Errorf("%s runs hooks/bootstrap.sh as a salvage but does not declare \"timeout\": %d", event, salvageTimeout)
				}
				if msg := strings.TrimSpace(h.StatusMessage); msg == "" || !strings.HasPrefix(msg, "abcd") {
					t.Errorf("%s runs hooks/bootstrap.sh as a salvage with no statusMessage naming abcd, so its wait is a silent stall", event)
				}
			}
		}
	}
	slices.Sort(salvage)
	if want := []string{"PreCompact", "PreToolUse", "UserPromptSubmit"}; !slices.Equal(salvage, want) {
		t.Errorf("the salvage entries are %v, want %v — a hook that starts or stops salvaging updates the timeout table too", salvage, want)
	}
}
