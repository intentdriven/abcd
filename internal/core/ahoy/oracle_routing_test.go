package ahoy

import (
	"fmt"
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
// step renders abcd's proposal in counts (how many agents, how many at each
// tier, and their fan-out bounds: the product thinker's 2026-10-03 ruling on
// iss-2610031236155833, since a row per agent cannot fit one question), and on
// consent writes the full table under ~/.abcd/ (owner-only), then offers the
// repository file in a separate question and writes it on consent. What it
// writes is exactly the bundled proposal, read back through the resolver every
// delegating verb uses.
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
			roster, proposal := oracle.Roster(), oracle.Proposal()
			perTier := map[oracle.Tier]int{}
			for _, agent := range roster {
				perTier[proposal[agent].Tier]++
			}
			want := []string{fmt.Sprintf("%d agents: ", len(roster)), "~/.abcd/oracle-routing.json"}
			for tier, n := range perTier {
				want = append(want, fmt.Sprintf("%d %s", n, tier))
			}
			for _, w := range want {
				if !strings.Contains(q, w) {
					t.Errorf("the machine offer does not say %q:\n%s", w, q)
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
	want := []GapCategory{Dependency, SafeAutocreate, ConfigChange, StatusLine, OracleRouting, DrainRule, UserState, PluginOwned}
	if !reflect.DeepEqual(categoryPromptOrder, want) {
		t.Fatalf("categoryPromptOrder = %v, want %v", categoryPromptOrder, want)
	}
}

// TestOracleRoutingRepoWriteNamesASymlinkLeavingTheRepository: a repository
// whose .abcd/config is a symlink out of the checkout gets no routing file
// written through it, and the refusal names the symlink rather than claiming
// the file appeared while the question was open.
func TestOracleRoutingRepoWriteNamesASymlinkLeavingTheRepository(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	outside := t.TempDir()
	link := filepath.Join(repo, ".abcd", "config")
	if err := os.RemoveAll(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	res, err := Install(repo, InstallOptions{}, routingPrompter(false, true))
	if err != nil {
		t.Fatal(err)
	}
	notes := strings.Join(res.Notes, "\n")
	if !strings.Contains(notes, ".abcd/config is a symlink") || strings.Contains(notes, "appeared while the question was open") {
		t.Fatalf("notes %q", notes)
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("a routing file was written through the symlink: %v", entries)
	}
}

// TestProposalCountsSayTheProposalInCounts is the form the machine offer says
// the proposal in (iss-2610031236155833): the number of agents, the number at
// each tier with the reason that tier is proposed, in the tier vocabulary
// reversed (frontier before economy), and
// the fan-out bounds, as one figure where every agent shares it and in counts
// where they differ, never as a list of agents.
func TestProposalCountsSayTheProposalInCounts(t *testing.T) {
	// Names no copy of the counts could contain by accident, so the check
	// that no agent is named below cannot pass vacuously.
	roster := []string{"agent-a", "agent-b", "agent-c", "agent-d"}
	same := oracle.Table{
		"agent-a": {Tier: oracle.Economy, FanOut: 1}, "agent-b": {Tier: oracle.Frontier, FanOut: 1},
		"agent-c": {Tier: oracle.Economy, FanOut: 1}, "agent-d": {Tier: oracle.Economy, FanOut: 1},
	}
	if got, want := proposalCounts(roster, same),
		"4 agents: 1 frontier, "+routingTierReason[oracle.Frontier]+"; 3 economy, "+routingTierReason[oracle.Economy]+"; fan-out 1 each"; got != want {
		t.Errorf("one shared fan-out:\n got %q\nwant %q", got, want)
	}
	differ := oracle.Table{
		"agent-a": {Tier: oracle.Economy, FanOut: 3}, "agent-b": {Tier: oracle.Economy, FanOut: 1},
		"agent-c": {Tier: oracle.Economy, FanOut: 1}, "agent-d": {Tier: oracle.Economy, FanOut: 3},
	}
	if got, want := proposalCounts(roster, differ),
		"4 agents: 4 economy, "+routingTierReason[oracle.Economy]+"; fan-out 1 for 2, 3 for 2"; got != want {
		t.Errorf("fan-outs that differ:\n got %q\nwant %q", got, want)
	}
	for _, table := range []oracle.Table{same, differ} {
		for _, agent := range roster {
			if counts := proposalCounts(roster, table); strings.Contains(counts, agent) {
				t.Errorf("the counts name the agent %q: %q", agent, counts)
			}
		}
	}
}

// TestMachineRoutingOfferAsksTheHarnessForTheTier holds the offer to what the
// system does for a step no provider serves (itd-2609170822093401 criterion 4):
// the step goes to the harness with its tier named in the request, and the host
// decides, so the offer says the harness is asked for the tier and never that
// the step runs at it, which no receipt proves (iss-2610031236155833).
func TestMachineRoutingOfferAsksTheHarnessForTheTier(t *testing.T) {
	q := machineRoutingQuestion()
	if !strings.Contains(q, "goes to the harness, asked for its tier") {
		t.Errorf("the offer does not say the harness is asked for the tier: %q", q)
	}
	if strings.Contains(q, "at its tier") {
		t.Errorf("the offer promises the step runs at its tier: %q", q)
	}
}

// TestEveryProposedTierSaysWhyItIsProposed keeps the counts honest: the offer
// says why each tier it counts is proposed, so a tier the bundled proposal
// starts to use fails here until its reason is written.
func TestEveryProposedTierSaysWhyItIsProposed(t *testing.T) {
	for agent, r := range oracle.Proposal() {
		if routingTierReason[r.Tier] == "" {
			t.Errorf("%s is proposed at %s, which the offer gives no reason for", agent, r.Tier)
		}
	}
}
