package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/intentdriven/abcd/internal/core/runner"
)

// manifestVerbRe is how a hook script names each verb it hands its event to.
var manifestVerbRe = regexp.MustCompile(`"\$r/abcd" ((?:hook|guard) [a-z][a-z-]*)`)

// TestSessionHooksWireEveryManifestEventToTheSameVerb: a session's hooks are the plugin
// manifest's, event by event, matcher by matcher, verb by verb and in the same order. The
// one event the manifest may lack is the one at a sub-agent's start, and where it holds it
// the two agree.
func TestSessionHooksWireEveryManifestEventToTheSameVerb(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoHooksDir(t), "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Hooks) == 0 {
		t.Fatal("the manifest wires nothing")
	}
	table := map[string]runner.SessionHook{}
	for _, h := range runner.SessionHooks() {
		if _, twice := table[h.Event]; twice {
			t.Errorf("%s is in the session's table twice", h.Event)
		}
		table[h.Event] = h
	}
	for event, groups := range manifest.Hooks {
		if len(groups) != 1 {
			t.Errorf("%s: the manifest holds %d groups, and the session's table one for each event", event, len(groups))
			continue
		}
		var verbs []string
		for _, h := range groups[0].Hooks {
			for _, m := range manifestVerbRe.FindAllStringSubmatch(resolveHookCommand(t, h.Command), -1) {
				verbs = append(verbs, m[1])
			}
		}
		if len(verbs) == 0 {
			t.Errorf("%s: no verb is read from the manifest's command", event)
		}
		h, held := table[event]
		switch {
		case !held:
			t.Errorf("%s: the manifest wires it to %q, and a session does not", event, verbs)
		case h.Matcher != groups[0].Matcher || !slices.Equal(h.Verbs, verbs):
			t.Errorf("%s: the manifest has %q %q, a session %q %q", event, groups[0].Matcher, verbs, h.Matcher, h.Verbs)
		}
	}
	for event := range table {
		if _, held := manifest.Hooks[event]; !held && event != "SubagentStart" {
			t.Errorf("%s: a session wires it, and the manifest does not", event)
		}
	}
}
