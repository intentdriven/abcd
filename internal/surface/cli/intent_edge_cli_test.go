package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// intent_edge_cli_test.go — iss-2610040758579292. `abcd intent edge` writes
// and removes an intent's blocked_by and builds_on edges, the counterpart of
// `abcd capture link`: the edge is written, an id no intent store holds is
// refused on exit 2 with nothing written, and the JSON carries both lists.
func TestIntentEdgeAtTheCLI(t *testing.T) {
	repo := intentTestRepo(t)
	rel := cliDrafts + "/itd-10-alpha.md"
	writeRepoFile(t, repo, rel, cliDraftWithAC("itd-10", "alpha"))
	writeRepoFile(t, repo, cliDrafts+"/itd-27-beta.md", cliDraftWithAC("itd-27", "beta"))
	writeRepoFile(t, repo, cliPlanned+"/itd-2-gamma.md", "---\nid: itd-2\nslug: gamma\nspec_id: spc-1\nkind: standalone\n---\n# gamma\n")

	out := runCLI(t, "intent", "edge", "itd-10", "--blocked-by", "itd-27", "--builds-on", "itd-2", "--json")
	var res struct {
		IntentID  string   `json:"intent_id"`
		Path      string   `json:"path"`
		BlockedBy []string `json:"blocked_by"`
		BuildsOn  []string `json:"builds_on"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("intent edge --json: %v\n%s", err, out)
	}
	if res.IntentID != "itd-10" || res.Path != rel || strings.Join(res.BlockedBy, ",") != "itd-27" || strings.Join(res.BuildsOn, ",") != "itd-2" {
		t.Fatalf("intent edge payload = %+v", res)
	}
	data, _ := os.ReadFile(filepath.Join(repo, rel))
	if !strings.Contains(string(data), "\nblocked_by: [itd-27]\n") || !strings.Contains(string(data), "\nbuilds_on: [itd-2]\n") {
		t.Fatalf("both edges must be written:\n%s", data)
	}

	before := string(data)
	_, err := runCLIErr(t, "intent", "edge", "itd-10", "--blocked-by", "itd-99")
	if err == nil || exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "abcd intent edge") || !strings.Contains(err.Error(), "not found in the intent store") {
		t.Fatalf("an unknown id must refuse on exit 2: exit = %d (%v)", exitCodeOf(err), err)
	}
	if after, _ := os.ReadFile(filepath.Join(repo, rel)); string(after) != before {
		t.Fatal("a refused edge wrote the record")
	}

	text := string(runCLI(t, "intent", "edge", "itd-10", "--unblock", "itd-27"))
	if !strings.Contains(text, "itd-10") || !strings.Contains(text, "blocked_by: []") || !strings.Contains(text, "builds_on: [itd-2]") {
		t.Fatalf("the plain render names both lists after the write: %q", text)
	}
}
