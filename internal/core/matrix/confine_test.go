package matrix

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// Each copy is confined to its own clone by what it can read of where it runs
// (the coordinator's ruling on step 8): every copy, a code, a writing or a
// check copy alike, is made under an alias minted at random, so the folder it
// works in and the folder above it name nothing but the run and aliases; the
// hidden tests sit under a name that spells no task; and the record of which
// alias is which copy lives in the run's state in the lab, never in the clone
// store.
func TestEveryCopyIsMadeUnderAnAliasAndTheStoreNamesNoTaskCellOrRoute(t *testing.T) {
	r, home, m, rec, store := identityLab(t)
	keys := []TrialKey{
		{Task: "feature-a", Cell: "or-cheap", Repeat: 1},
		{Task: "doc-page", Cell: "host-big", Repeat: 2},
		{Task: checkTask, Cell: "lr", Repeat: 1},
	}
	for _, key := range keys {
		if _, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key); err != nil {
			t.Fatalf("makeCopyClone(%s): %v", key, err)
		}
	}
	if _, err := placeHiddenTests(store, rec.RunID, "feature-a", widgetHidden); err != nil {
		t.Fatal(err)
	}
	alias := regexp.MustCompile(`^copy_[0-9a-f]{32}$`)
	names := func(dir string) []string {
		t.Helper()
		ents, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, e := range ents {
			out = append(out, e.Name())
		}
		return out
	}
	if got := names(filepath.Join(store, rec.RunID)); len(got) != len(keys) {
		t.Fatalf("the run's folder in the store holds %v; want one alias per copy", got)
	} else {
		for _, n := range got {
			if !alias.MatchString(n) {
				t.Errorf("the run's folder in the store holds %q, which is no alias", n)
			}
		}
	}
	runIdentityWords := []string{"feature-a", "feature", "doc-page", checkTask, "or-cheap", "host-big", "opencode", "openrouter", "vendor", "qwen"}
	err := filepath.WalkDir(store, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(store, p)
		// What a copy reads above its clone: every folder of the store that is
		// no clone's own content.
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) >= 3 && parts[0] == rec.RunID {
			return filepath.SkipDir
		}
		for _, w := range runIdentityWords {
			if strings.Contains(rel, w) {
				t.Errorf("the store names %q at %s, readable from where a copy runs", w, rel)
			}
		}
		if slices.Contains(parts, "lr") || slices.Contains(parts, "1") || slices.Contains(parts, "2") {
			t.Errorf("the store names a short cell id or a repeat at %s", rel)
		}
		if !d.IsDir() && len(parts) < 4 {
			data, err := os.ReadFile(p)
			if err == nil && strings.Contains(string(data), "alias") {
				t.Errorf("the store holds an alias record at %s", rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		rel := filepath.Join(runDir(home, m, rec.RunID), "aliases", key.Task, key.Cell, "1.json")
		if key.Repeat == 2 {
			rel = filepath.Join(runDir(home, m, rec.RunID), "aliases", key.Task, key.Cell, "2.json")
		}
		if _, err := os.Stat(rel); err != nil {
			t.Errorf("copy %s's alias is not recorded in the run's state in its lab: %v", key, err)
		}
	}
	if under := strings.HasPrefix(runDir(home, m, rec.RunID), store); under {
		t.Fatal("the run's state is inside the clone store")
	}
}

// A copy that reads the run's state, where the alias record and every other
// record of the run sit, breaks own_copy_only: the record names its cell, so
// reading it un-blinds the copy.
func TestACopyThatReadsTheRunsStateIsExcluded(t *testing.T) {
	c := newComplianceCase(t)
	state := abcdhome.Path(os.Getenv("HOME"), "lab", strings.Repeat("e", 40))
	c.in.State = state
	for name, line := range map[string]string{
		"an absolute path": "cat " + filepath.Join(state, "lab-1", "state", "matrix", cloneRun, "aliases", "widget", "cheap", "1.json"),
		"through home":     "ls ~/" + strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(state, os.Getenv("HOME"))), "/"),
		"JSON-escaped":     `{"path":"` + strings.ReplaceAll(state, "/", `\/`) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			in := c.in
			in.Transcript = append(append([]byte{}, c.in.Transcript...), []byte(line+"\n")...)
			got, err := checkCompliance(in)
			if err != nil {
				t.Fatal(err)
			}
			if ch := ruleOf(t, got, RuleOwnCopyOnly); ch.State != StateBroken || !strings.Contains(ch.Detail, "run's state") {
				t.Fatalf("own_copy_only = %+v; want it broken, naming the run's state", ch)
			}
		})
	}
}

// The record of which alias is which copy, and a refusal that names a copy's
// key, never reach a copy's judges: a failed check copy whose runner's words
// carry an error naming its key, its cell's id under three bytes included,
// is recorded with the key withheld, and neither the judging packet of a
// writing task nor its key's documents carry an alias or the alias record's
// path.
func TestTheAliasRecordAndAKeyInAnErrorNeverReachAJudge(t *testing.T) {
	r, _, m, rec, store := identityLab(t)
	gates, err := snapshotGates(r.Root())
	if err != nil {
		t.Fatal(err)
	}
	key := TrialKey{Task: checkTask, Cell: "lr", Repeat: 2}
	if _, err := makeCopyClone(m.ID, r.Root(), store, rec.RunID, rec.FrozenCommit, key); err != nil {
		t.Fatal(err)
	}
	in := checkInputs(t, r, rec, store, key, CheckDocFidelity, gates, nil)
	in.Failure = "copy " + key.String() + "'s clone was not made: " + aliasRel(rec.RunID, key) + " and trials/" + key.Task + "/" + key.Cell + "/2.json"
	fc, err := collectCheckCopy(m.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"lr/2", "/lr", key.String()} {
		if strings.Contains(fc.Reason, leak) {
			t.Errorf("a failed copy's reason carries %q: %s", leak, fc.Reason)
		}
	}

	w := writingInputs(2, 2, 3)
	for i := range w.Copies {
		w.Copies[i].Alias = "copy_" + strings.Repeat(string("0123456789abcdef"[i]), 32)
		w.Copies[i].Text += "Error: copy " + w.Copies[i].Key.String() + " at " + aliasRel(w.Run, w.Copies[i].Key) + ", alias " + w.Copies[i].Alias + ".\n"
	}
	p, _ := draw(t, w, seedA)
	pj, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range w.Copies {
		for _, leak := range []string{c.Alias, c.Key.String(), c.Key.Cell} {
			if strings.Contains(string(pj), leak) {
				t.Errorf("the writing packet carries %q", leak)
			}
		}
	}
}
