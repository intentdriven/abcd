package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `abcd intent plan itd-A itd-B --bundle <name>` is the bundle command's CLI
// door (itd-34): both drafts reach planned/ on one shared spec.
func TestIntentPlanBundleCLI(t *testing.T) {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	writeRepoFile(t, repo, cliDrafts+"/itd-11-beta.md", cliDraftWithAC("itd-11", "beta"))

	out := runCLI(t, "intent", "plan", "itd-10", "itd-11", "--bundle", "alpha-beta", "--json")
	var got struct {
		Bundle string `json:"bundle"`
		Spec   struct {
			ID      string   `json:"id"`
			Intents []string `json:"intents"`
		} `json:"spec"`
		Members []struct {
			Intent struct {
				Bucket string `json:"bucket"`
				Bundle string `json:"bundle"`
			} `json:"intent"`
		} `json:"members"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("plan --json not JSON: %v\n%s", err, out)
	}
	if got.Bundle != "alpha-beta" || len(got.Spec.Intents) != 2 || len(got.Members) != 2 {
		t.Fatalf("bundle plan result = %+v", got)
	}
	for _, rel := range []string{"itd-10-alpha.md", "itd-11-beta.md"} {
		if _, err := os.Stat(filepath.Join(repo, cliPlanned, rel)); err != nil {
			t.Fatalf("%s must be planned: %v", rel, err)
		}
	}

	text := string(runCLI(t, "intent", "plan", "--help"))
	if !strings.Contains(text, "--bundle") {
		t.Fatalf("plan --help must document --bundle:\n%s", text)
	}
}

// On the CLI the bundle name is refused absent, and a name given for one
// intent is refused too: nothing moves either way.
func TestIntentPlanBundleCLIRefusesWithoutAName(t *testing.T) {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	writeRepoFile(t, repo, cliDrafts+"/itd-11-beta.md", cliDraftWithAC("itd-11", "beta"))

	_, err := runCLIErr(t, "intent", "plan", "itd-10", "itd-11")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "--bundle") {
		t.Fatalf("several intents without --bundle must exit 2 naming the flag: %v", err)
	}
	if _, err := runCLIErr(t, "intent", "plan", "itd-10", "--bundle", "solo"); exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "two or more intents") {
		t.Fatalf("--bundle on one intent must exit 2, saying a bundle has two or more: %v", err)
	}
	for _, rel := range []string{"itd-10-alpha.md", "itd-11-beta.md"} {
		if _, err := os.Stat(filepath.Join(repo, cliDrafts, rel)); err != nil {
			t.Fatalf("%s must stay a draft: %v", rel, err)
		}
	}
}

// `abcd intent reclassify` is the reclassify verb's CLI door: a supersession
// prints the paths it moved and wrote.
func TestIntentReclassifyCLI(t *testing.T) {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	writeRepoFile(t, repo, cliDrafts+"/itd-20-successor.md", cliDraftWithAC("itd-20", "successor"))

	out := string(runCLI(t, "intent", "reclassify", "itd-10", "--kind", "superseded", "--by", "itd-20", "--reason", "absorbed by itd-20"))
	for _, want := range []string{
		"abcd intent reclassify — itd-10 standalone -> superseded",
		"moved: " + cliDrafts + "/itd-10-alpha.md -> .abcd/development/intents/superseded/itd-10-alpha.md",
		"wrote: " + cliDrafts + "/itd-20-successor.md",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("reclassify output missing %q:\n%s", want, out)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".abcd/development/intents/superseded/itd-10-alpha.md")); err != nil {
		t.Fatalf("the record must be superseded: %v", err)
	}
}

// A refusal exits 2 and names the remedy.
func TestIntentReclassifyCLIRefusesAShippedDiscipline(t *testing.T) {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, ".abcd/development/intents/shipped/itd-10-alpha.md",
		"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\nimpact: fix\n---\n# alpha\n")
	_, err := runCLIErr(t, "intent", "reclassify", "itd-10", "--kind", "discipline")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "file a discipline that supersedes it") {
		t.Fatalf("a shipped intent to discipline must exit 2 naming the remedy: %v", err)
	}
}

// `abcd spec close` on a bundle's shared spec names every member it shipped.
func TestSpecCloseShipsABundleCLI(t *testing.T) {
	repo := intentTestRepo(t)
	writeRepoFile(t, repo, cliDrafts+"/itd-10-alpha.md", cliDraftWithAC("itd-10", "alpha"))
	writeRepoFile(t, repo, cliDrafts+"/itd-11-beta.md", cliDraftWithAC("itd-11", "beta"))
	var planned struct {
		Spec struct {
			ID string `json:"id"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(runCLI(t, "intent", "plan", "itd-10", "itd-11", "--bundle", "alpha-beta", "--json"), &planned); err != nil {
		t.Fatal(err)
	}
	out := string(runCLI(t, "spec", "close", planned.Spec.ID, "--impact", "additive"))
	for _, want := range []string{"reconciled intent itd-10: planned -> shipped", "reconciled intent itd-11: planned -> shipped"} {
		if !strings.Contains(out, want) {
			t.Fatalf("spec close output missing %q:\n%s", want, out)
		}
	}
}
