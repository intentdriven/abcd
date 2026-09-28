package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// TestDogfoodingRecallNamesEveryTopLevelVerb holds this repository's
// DOGFOODING domain to its documented contract (iss-2609251357387577):
// AGENTS.md says the go-run rule is injected on a prompt naming abcd or any of
// its top-level verbs, the Available Commands list of `abcd --help`. The recall
// list is hand-kept, so without this reader a verb added to the tree is a verb
// whose prompts never receive the rule — which is how seven of them drifted out.
func TestDogfoodingRecallNamesEveryTopLevelVerb(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(testRepoRoot(), ".abcd", "rules.json"))
	if err != nil {
		t.Fatal(err)
	}
	var rules struct {
		Domains map[string]struct {
			Recall []string `json:"recall"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(data, &rules); err != nil {
		t.Fatalf("parse .abcd/rules.json: %v", err)
	}
	dom, ok := rules.Domains["DOGFOODING"]
	if !ok {
		t.Fatal(".abcd/rules.json declares no DOGFOODING domain")
	}
	root := NewRootCommand()
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()
	for _, c := range root.Commands() {
		if !c.IsAvailableCommand() && c.Name() != "help" {
			continue
		}
		if !slices.Contains(dom.Recall, c.Name()) {
			t.Errorf("top-level verb %q is in `abcd --help` but not in the DOGFOODING domain's recall list in .abcd/rules.json", c.Name())
		}
	}
}
