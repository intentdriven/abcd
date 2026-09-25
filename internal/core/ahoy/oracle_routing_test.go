package ahoy

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// routingPrompter approves every category question and answers the two
// routing offers as told, recording every question asked. Every other offer
// (the status line, the identity pin) is declined, so only the routing files
// can change.
func routingPrompter(machine, repo bool) *scriptedPrompter {
	return &scriptedPrompter{confirm: func(q string) bool {
		switch {
		case strings.HasPrefix(q, "Apply "):
			return true
		case strings.Contains(q, oracleRoutingMachineQuestionTail):
			return machine
		case strings.Contains(q, oracleRoutingRepoQuestionTail):
			return repo
		}
		return false
	}}
}

func routingPaths(home, repo string) (machine, repoFile string) {
	return filepath.Join(home, ".abcd", filepath.FromSlash(layered.OracleRouting.MachineRel)),
		filepath.Join(repo, filepath.FromSlash(layered.OracleRouting.RepoRel))
}

// TestOracleRoutingConsentWritesTheProposal is AC 2's consent half: the install
// step renders abcd's proposed table, one row per agent with its tier and
// fan-out bound, and on consent writes it under ~/.abcd/ (owner-only), then
// offers the repository file in a separate question and writes it on consent.
// What it writes is exactly the bundled proposal, read back through the
// resolver every delegating verb uses.
func TestOracleRoutingConsentWritesTheProposal(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := installedRepo(t)
	machinePath, repoPath := routingPaths(home, repo)

	p := routingPrompter(true, true)
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}

	var machineQ, repoQ = -1, -1
	for i, q := range p.asked {
		if strings.Contains(q, oracleRoutingMachineQuestionTail) {
			machineQ = i
			for _, agent := range oracle.Roster() {
				tier := oracle.Proposal()[agent].Tier
				if !strings.Contains(q, agent) || !strings.Contains(q, string(tier)) {
					t.Errorf("the machine offer does not render %s at %s:\n%s", agent, tier, q)
				}
			}
		}
		if strings.Contains(q, oracleRoutingRepoQuestionTail) {
			repoQ = i
		}
	}
	if machineQ < 0 || repoQ < 0 || repoQ < machineQ {
		t.Fatalf("want the machine offer, then a separate repository offer; asked %q", p.asked)
	}

	fi, err := os.Stat(machinePath)
	if err != nil {
		t.Fatalf("the machine table was not written: %v (writes %v, notes %v)", err, res.Writes, res.Notes)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("the machine table is %v, want 0600: its reader refuses a file others can write", fi.Mode().Perm())
	}
	if _, err := os.Stat(repoPath); err != nil {
		t.Fatalf("the repository table was not written: %v", err)
	}

	for _, roots := range []layered.Roots{{Home: home}, {Repo: repo, Home: t.TempDir()}} {
		l, err := oracle.Load(roots)
		if err != nil {
			t.Fatalf("the written table does not load: %v", err)
		}
		if !l.Accepted() || len(l.Diagnostics) != 0 {
			t.Fatalf("accepted %v, diagnostics %v", l.Accepted(), l.Diagnostics)
		}
		for _, agent := range oracle.Roster() {
			r, err := oracle.Resolve(agent, l, oracle.NoConnections{})
			if err != nil {
				t.Fatal(err)
			}
			want := oracle.Proposal()[agent]
			if r.Row.Tier != want.Tier || r.Row.FanOut != want.FanOut || r.Source == layered.Bundled {
				t.Errorf("%s resolves to %+v from %s, want the written proposal row %+v", agent, r.Row, r.Source, want)
			}
		}
	}

	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, OracleRoutingMachineGapID) || hasGap(det.Gaps, OracleRoutingRepoGapID) {
		t.Error("an accepted table is offered again")
	}
}

// TestOracleRoutingDeclineWritesNothing is AC 2's refusal half: declining
// writes nothing and is not persisted, so the next install offers again; the
// two offers are independent.
func TestOracleRoutingDeclineWritesNothing(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := installedRepo(t)
	machinePath, repoPath := routingPaths(home, repo)

	if _, err := Install(repo, InstallOptions{}, routingPrompter(false, false)); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{machinePath, repoPath} {
		if _, err := os.Lstat(p); err == nil {
			t.Errorf("a declined offer wrote %s", p)
		}
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !hasGap(det.Gaps, OracleRoutingMachineGapID) || !hasGap(det.Gaps, OracleRoutingRepoGapID) {
		t.Error("a decline was persisted; both offers must stand for the next install")
	}

	if _, err := Install(repo, InstallOptions{}, routingPrompter(true, false)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(machinePath); err != nil {
		t.Errorf("the machine offer accepted alone was not written: %v", err)
	}
	if _, err := os.Lstat(repoPath); err == nil {
		t.Error("declining the repository offer wrote the repository table")
	}
	det, _ = Detect(repo)
	if hasGap(det.Gaps, OracleRoutingMachineGapID) || !hasGap(det.Gaps, OracleRoutingRepoGapID) {
		t.Errorf("after accepting only the machine offer, gaps = %v", det.Gaps)
	}
}

// TestOracleRoutingYesSkipsBothAndSaysSo: --yes approves the category but
// writes neither table, and reports both under optional_skipped, the same
// shape as the status line.
func TestOracleRoutingYesSkipsBothAndSaysSo(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, installOpts(), RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{OracleRoutingMachineGapID, OracleRoutingRepoGapID} {
		if !containsString(res.OptionalSkipped, id) {
			t.Errorf("optional_skipped = %v, want it to name %s", res.OptionalSkipped, id)
		}
	}
	machinePath, repoPath := routingPaths(home, repo)
	for _, p := range []string{machinePath, repoPath} {
		if _, err := os.Lstat(p); err == nil {
			t.Errorf("--yes wrote %s", p)
		}
	}
}

// TestUninstallLeavesTheRoutingTables: the tables are the user's
// configuration, so uninstall leaves both.
func TestUninstallLeavesTheRoutingTables(t *testing.T) {
	home, _ := setupHermetic(t)
	repo := installedRepo(t)
	if _, err := Install(repo, InstallOptions{}, routingPrompter(true, true)); err != nil {
		t.Fatal(err)
	}
	if _, err := Uninstall(repo, ""); err != nil {
		t.Fatal(err)
	}
	machinePath, repoPath := routingPaths(home, repo)
	for _, p := range []string{machinePath, repoPath} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("uninstall removed %s: %v", p, err)
		}
	}
}

// TestOracleRoutingIsAskedAfterTheStatusLine pins the consent order the spec
// names: …, status-line, oracle-routing, user-state, …
func TestOracleRoutingIsAskedAfterTheStatusLine(t *testing.T) {
	want := []GapCategory{Dependency, SafeAutocreate, ConfigChange, StatusLine, OracleRouting, UserState, PluginOwned}
	if !reflect.DeepEqual(categoryPromptOrder, want) {
		t.Fatalf("categoryPromptOrder = %v, want %v", categoryPromptOrder, want)
	}
}
