package cli

import (
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
)

// The per-event timeouts in hooks/hooks.json are a ruling, not a default to
// drift from (iss-323; the product thinker's rulings of 2026-09-29, J25 as
// corrected by CH1 and CH2). The every-message hook, UserPromptSubmit, runs
// hooks/bootstrap.sh as a salvage and declares 120 seconds, because the host's
// default for that event is shorter than bootstrap's download worst case and a
// killed salvage stamps .bootstrap.attempt and suppresses the retry for ten
// minutes. The before-command (PreToolUse) and before-compaction (PreCompact)
// salvage hooks declare no timeout and take the host's default of ten minutes,
// so a slow first download always finishes there. SessionStart, the primary
// provisioner, declares 240. The transcript hooks declare none. The wait must
// also never be silent: every entry that runs bootstrap.sh, SessionStart
// included, names a statusMessage, the host's spinner text shown while a hook
// runs, because the salvage command itself sends bootstrap's output to
// /dev/null. One message serves entries whose limits are two, four and ten
// minutes, so it promises no duration.

// wantHookTimeouts pins every event the manifest declares and the timeout its
// one handler carries; 0 means the field is absent. An event added, removed or
// re-timed without updating this table fails the test.
var wantHookTimeouts = map[string]int{
	"SessionStart":     240,
	"UserPromptSubmit": 120,
	"PreToolUse":       0,
	"PreCompact":       0,
	"SessionEnd":       0,
	"SubagentStop":     0,
}

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

// TestEverySalvageEntryShowsItsWait derives the entries from the manifest
// rather than from a list: every entry whose command runs hooks/bootstrap.sh,
// SessionStart included, shows a spinner message naming abcd, so a new
// provisioning entry cannot arrive silent. The message states no duration,
// because no one duration is true for every entry that shows it.
func TestEverySalvageEntryShowsItsWait(t *testing.T) {
	doc := decodedHooksManifest(t)
	var provisioning []string
	for event, entries := range doc.Hooks {
		for _, entry := range entries {
			for _, h := range entry.Hooks {
				if !strings.Contains(h.Command, "hooks/bootstrap.sh") {
					continue
				}
				provisioning = append(provisioning, event)
				msg := strings.TrimSpace(h.StatusMessage)
				if msg == "" || !strings.HasPrefix(msg, "abcd") {
					t.Errorf("%s runs hooks/bootstrap.sh with no statusMessage naming abcd, so its wait is a silent stall", event)
					continue
				}
				if f := durationPromise(msg); f != "" {
					t.Errorf("%s shows %q, which promises a duration (%s) that is not true for every entry carrying it", event, msg, f)
				}
			}
		}
	}
	slices.Sort(provisioning)
	if want := []string{"PreCompact", "PreToolUse", "SessionStart", "UserPromptSubmit"}; !slices.Equal(provisioning, want) {
		t.Errorf("the entries that run hooks/bootstrap.sh are %v, want %v — a hook that starts or stops provisioning updates the timeout table too", provisioning, want)
	}
}

// durationPromise names the first word of msg that states a length of time,
// or returns "" when it states none.
func durationPromise(msg string) string {
	for _, w := range strings.FieldsFunc(strings.ToLower(msg), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		switch {
		case strings.IndexFunc(w, func(r rune) bool { return r >= '0' && r <= '9' }) >= 0:
			return w
		case strings.HasPrefix(w, "second"), strings.HasPrefix(w, "minute"), strings.HasPrefix(w, "hour"):
			return w
		}
	}
	return ""
}

func TestDurationPromiseSeesAStatedDuration(t *testing.T) {
	for msg, want := range map[string]string{
		"abcd: a first run downloads it, which can take up to two minutes": "minutes",
		"abcd: this can take 120s":                                   "120s",
		"abcd: checking the plugin binary; a first run downloads it": "",
	} {
		if got := durationPromise(msg); got != want {
			t.Errorf("durationPromise(%q) = %q, want %q", msg, got, want)
		}
	}
}
