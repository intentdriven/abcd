package rules

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// subagentFactFiles lists the leaves under the state folder d.
func subagentFactFiles(t *testing.T, d string) []string {
	t.Helper()
	entries, err := os.ReadDir(d)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// TestSubagentFactsAreKeptPerSessionAndAgent: a fact is keyed by the session
// and the agent together, written once at mode 0600, untouched by the prompt
// router's own state, and removed by PruneState at its age.
func TestSubagentFactsAreKeptPerSessionAndAgent(t *testing.T) {
	d := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", d)
	t.Setenv("HOME", t.TempDir())

	if _, ok, err := ReadSubagentFact("s1", "a1", SubagentTold); err != nil || ok {
		t.Fatalf("a fact never written reads ok=%v err=%v", ok, err)
	}
	told := SubagentRecord{File: "AGENTS.md", Root: "/w", By: SubagentToldByStart}
	if err := WriteSubagentFact("s1", "a1", SubagentTold, told); err != nil {
		t.Fatal(err)
	}
	got, ok, err := ReadSubagentFact("s1", "a1", SubagentTold)
	if err != nil || !ok || got.File != "AGENTS.md" || got.Root != "/w" || got.By != SubagentToldByStart {
		t.Fatalf("told = %+v ok=%v err=%v", got, ok, err)
	}
	// Another session, another agent and another fact see nothing of it. The
	// NUL between the parts keeps "s1"+"a1" from meeting "s1a"+"1".
	for _, k := range [][2]string{{"s2", "a1"}, {"s1", "a2"}, {"s1a", "1"}} {
		if _, ok, _ := ReadSubagentFact(k[0], k[1], SubagentTold); ok {
			t.Errorf("the fact of s1/a1 is read for %s/%s", k[0], k[1])
		}
	}
	for _, f := range []SubagentFact{SubagentPassed, SubagentHeld, SubagentUnjudged, SubagentJudged} {
		if _, ok, _ := ReadSubagentFact("s1", "a1", f); ok {
			t.Errorf("the told fact is read as %s", f)
		}
	}
	// Written once: a second write leaves the first as it stands.
	if err := WriteSubagentFact("s1", "a1", SubagentTold, SubagentRecord{By: SubagentToldByGuard}); err != nil {
		t.Fatal(err)
	}
	if again, _, _ := ReadSubagentFact("s1", "a1", SubagentTold); again.By != SubagentToldByStart || again.File != "AGENTS.md" {
		t.Errorf("a second write replaced the fact: %+v", again)
	}
	files := subagentFactFiles(t, d)
	if len(files) != 1 || !strings.HasSuffix(files[0], ".told") || strings.Contains(files[0], "s1") {
		t.Fatalf("state folder holds %v; want the one hashed .told leaf", files)
	}
	path := filepath.Join(d, files[0])
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("the fact's mode = %v, want 0600", fi.Mode().Perm())
	}
	// The reset hook's removal of the session's ledger leaves the fact.
	if err := ResetState("s1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := ReadSubagentFact("s1", "a1", SubagentTold); !ok {
		t.Error("resetting the session's ledger dropped its sub-agent's fact")
	}
	PruneState(StateTTL)
	if _, ok, _ := ReadSubagentFact("s1", "a1", SubagentTold); !ok {
		t.Error("the sweep removed a fact under its age")
	}
	old := time.Now().Add(-StateTTL - time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	PruneState(StateTTL)
	if _, ok, err := ReadSubagentFact("s1", "a1", SubagentTold); ok || err != nil {
		t.Errorf("a fact past its age is still read: ok=%v err=%v", ok, err)
	}
}

// TestSubagentToldFactTakesOnlyTheTwoFileNames: a told fact names one of the
// two conventions files or none. Anything else, and a fact that does not
// parse, one another account can write, or one that is a link, cannot be read.
func TestSubagentToldFactTakesOnlyTheTwoFileNames(t *testing.T) {
	d := t.TempDir()
	t.Setenv("ABCD_RULES_STATE_DIR", d)
	for _, file := range []string{"AGENTS.md", "CLAUDE.md", ""} {
		agent := "ok-" + file
		if err := WriteSubagentFact("s", agent, SubagentTold, SubagentRecord{File: file, By: SubagentToldByStart}); err != nil {
			t.Fatalf("%q: %v", file, err)
		}
		if got, ok, err := ReadSubagentFact("s", agent, SubagentTold); err != nil || !ok || got.File != file {
			t.Errorf("%q: read %+v ok=%v err=%v", file, got, ok, err)
		}
	}
	plant := func(agent, body string, mode os.FileMode) string {
		t.Helper()
		p := filepath.Join(d, subagentLeaf("s", agent, SubagentTold))
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(p, mode); err != nil {
			t.Fatal(err)
		}
		return p
	}
	plant("other", `{"file":"README.md","by":"start"}`, 0o600)
	plant("path", `{"file":"../AGENTS.md","by":"start"}`, 0o600)
	plant("who", `{"file":"AGENTS.md","by":"someone"}`, 0o600)
	plant("garbage", `{not json`, 0o600)
	plant("writable", `{"file":"AGENTS.md","by":"start"}`, 0o622)
	target := plant("target", `{"file":"AGENTS.md","by":"start"}`, 0o600)
	if err := os.Symlink(target, filepath.Join(d, subagentLeaf("s", "link", SubagentTold))); err != nil {
		t.Fatal(err)
	}
	for _, agent := range []string{"other", "path", "who", "garbage", "writable", "link"} {
		if got, ok, err := ReadSubagentFact("s", agent, SubagentTold); err == nil || ok {
			t.Errorf("%s: read %+v ok=%v err=%v; want it refused", agent, got, ok, err)
		}
	}
	// A folder that cannot be opened is told apart from one fact that cannot
	// be read: with the first nothing is known of any sub-agent.
	t.Setenv("ABCD_RULES_STATE_DIR", "")
	t.Setenv("HOME", "")
	if _, ok, err := ReadSubagentFact("s", "a", SubagentTold); ok || !errors.Is(err, ErrSubagentStore) {
		t.Errorf("no home: ok=%v err=%v, want ErrSubagentStore", ok, err)
	}
	if _, _, err := ReadSubagentFact("s", "other", SubagentTold); err == nil {
		t.Error("no home: the read gave no error")
	}
}
