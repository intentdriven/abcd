package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// TestImplementStreamsMaskTheTextARunStores (iss-2610121542240257): a run
// whose state.json stores a step title, a pending title, the pick's entry and
// an exclusion's reason, a record note, a sync path, the landing URL, a
// runner's reported model and a transcript carrying U+202E or U+009B is read,
// exit 0, by `implement record --json` and `implement status --json`, and
// neither stream carries the raw rune.
func TestImplementStreamsMaskTheTextARunStores(t *testing.T) {
	repo := buildRepo(t)
	secondBuildIntent(t, repo)
	var res struct {
		Start struct {
			RunID string `json:"run_id"`
		} `json:"start"`
	}
	if err := json.Unmarshal([]byte(mustImplement(t, "build", "next", "--json")), &res); err != nil {
		t.Fatal(err)
	}
	id := res.Start.RunID
	path := filepath.Join(repo.Root(), filepath.FromSlash(loop.StateRelPath(id)))
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"‮", "\u009b"} {
		var doc map[string]any
		if err := json.Unmarshal(good, &doc); err != nil {
			t.Fatal(err)
		}
		lane := doc["lanes"].([]any)[0].(map[string]any)
		lane["step_title"] = "Summary" + bad
		lane["syncs"] = []any{map[string]any{"at": "2026-10-12T10:00:00Z", "siblings": []any{}, "merged": "", "conflicted": true, "paths": []any{"a" + bad + ".md"}}}
		lane["landing"] = map[string]any{"closes": false, "records_done": false, "body_checked": false, "armed": false, "pr_url": "https://forge.example/1" + bad}
		lane["receipts"] = []any{map[string]any{"role": "implementer", "receipt": "lane-1/receipt.json", "model": "m",
			"route": map[string]any{"asked": "claude", "ran": "claude", "model": "claude" + bad}}}
		doc["pending"] = []any{map[string]any{"number": 2, "title": "two" + bad, "needs": []any{}}}
		pick := doc["pick"].(map[string]any)
		pick["entry"] = pick["entry"].(string) + bad
		pick["excluded"] = []any{map[string]any{"id": "itd-12", "check": "spec", "reason": "no spec" + bad}}
		doc["record"].([]any)[0].(map[string]any)["note"] = "started" + bad
		doc["transcripts"] = []any{map[string]any{"at": "2026-10-12T10:00:00Z", "path": "t" + bad, "session": "s", "stored": "x", "wrote": true}}
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"implement", "record", "--run", id, "--json"}, {"implement", "status", "--json"}} {
			code, out, errOut := implementCLI(t, args...)
			if strings.ContainsAny(out+errOut, "‮\u009b") {
				t.Fatalf("%q, %v: a raw rune reaches the stream: %q", bad, args, out)
			}
			if code != 0 || !strings.Contains(out, "Summary?") {
				t.Fatalf("%q, %v: the run is read, its text masked, exit 0: %d %s %s", bad, args, code, out, errOut)
			}
		}
	}
}
