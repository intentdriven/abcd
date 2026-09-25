package ahoy

// The oracle-routing consent `ahoy install` asks for (itd-2609170822093401
// AC 2, spc-2609180535002478 step 5). abcd ships its own proposal for the model
// tier and fan-out bound each agent deserves (internal/core/oracle), and
// nothing of it applies until it is accepted. The install step is where it is
// accepted:
//
//   - the machine offer renders the proposal as a table and, on consent,
//     writes it to ~/.abcd/oracle-routing.json, owner-only, because the
//     resolver refuses a machine file others can write;
//   - a second, separate offer writes the same table to the repository's
//     .abcd/config/oracle-routing.json, which is committed and wins over every
//     machine's table.
//
// Both are optional gaps in their own category, asked after the status line:
// --yes approves the category but writes neither (a routing table decides
// which model every delegated step asks for, so only an answered prompt may
// accept one) and reports both under optional_skipped. A decline writes
// nothing and records nothing, so the next install offers again. Uninstall
// leaves both files: they are the user's configuration.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/fsutil"
)

const (
	// OracleRoutingMachineGapID is the offer of the machine's routing table.
	OracleRoutingMachineGapID = "oracle_routing.machine_offered"
	// OracleRoutingRepoGapID is the offer of the repository's routing table.
	OracleRoutingRepoGapID = "oracle_routing.repo_offered"
)

// The two offers end on these questions, so a scripted answer stream and the
// transcript can tell them apart.
const (
	oracleRoutingMachineQuestionTail = "Accept abcd's proposed routing for this machine?"
	oracleRoutingRepoQuestionTail    = "Also write it to this repository's " + ".abcd/config/oracle-routing.json?"
)

// machineRoutingPath is ~/.abcd/oracle-routing.json, or "" when no home
// resolves.
func machineRoutingPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".abcd", filepath.FromSlash(layered.OracleRouting.MachineRel))
}

// absent reports whether nothing at all is at p: not a file, not a symlink,
// not a directory. Anything present is left to the resolver, which reads it
// or refuses it loudly; the offer is only ever for a table nobody has written.
func absent(p string) bool {
	_, err := os.Lstat(p)
	return errors.Is(err, fs.ErrNotExist)
}

// detectOracleRouting raises the two offers while their files are absent.
func detectOracleRouting(cwd string) []Gap {
	var gaps []Gap
	if p := machineRoutingPath(); p != "" && absent(p) {
		gaps = append(gaps, Gap{
			ID: OracleRoutingMachineGapID, Category: OracleRouting, Scope: "machine",
			Title:    "model-tier routing not accepted on this machine",
			Detail:   "abcd proposes a model tier and a fan-out bound for each of its agents; nothing of it applies until it is accepted, so every delegated step asks the harness for host-decides.",
			FixHint:  "ahoy install renders the proposal and writes ~/.abcd/oracle-routing.json only on consent; --yes never accepts it (run without --yes).",
			Required: false, Resolvable: true,
		})
	}
	if absent(filepath.Join(cwd, filepath.FromSlash(layered.OracleRouting.RepoRel))) {
		gaps = append(gaps, Gap{
			ID: OracleRoutingRepoGapID, Category: OracleRouting, Scope: "repo",
			Title:    "no model-tier routing committed for this repository",
			Detail:   "A committed .abcd/config/oracle-routing.json routes this repository's delegated steps for everyone who works in it, over each machine's own table.",
			FixHint:  "ahoy install offers abcd's proposal as the repository's table, separately from the machine's, and writes it only on consent; --yes never accepts it.",
			Required: false, Resolvable: true,
		})
	}
	return gaps
}

// stepOracleRouting makes the two offers, machine first, each on its own
// answer. It never runs under --yes.
func (a *applyCtx) stepOracleRouting() {
	if a.autoYes || !a.approved[OracleRouting] {
		return
	}
	if !a.has(OracleRoutingMachineGapID) && !a.has(OracleRoutingRepoGapID) {
		return
	}
	body, err := proposalTable()
	if err != nil {
		a.refuse("the model-tier routing was not offered: " + errText(err))
		return
	}
	if a.has(OracleRoutingMachineGapID) {
		if a.prompter.Confirm(machineRoutingQuestion()) {
			a.writeMachineRouting(body)
		}
	}
	if a.has(OracleRoutingRepoGapID) {
		if a.prompter.Confirm(repoRoutingQuestion()) {
			a.writeRepoRouting(body)
		}
	}
}

// writeMachineRouting writes the machine table 0600, refusing when anything
// appeared at the path since detection.
func (a *applyCtx) writeMachineRouting(body []byte) {
	p := machineRoutingPath()
	if p == "" {
		a.refuse("the model-tier routing was not written: the home directory could not be resolved, so ~/.abcd/oracle-routing.json has nowhere to go.")
		return
	}
	if !absent(p) {
		a.refuse("the model-tier routing was not written: ~/.abcd/oracle-routing.json appeared while the question was open, and it is left as it is.")
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		a.refuse("could not create ~/.abcd for the model-tier routing (" + errText(err) + "); nothing was written.")
		return
	}
	// 0600, never wider: the resolver refuses a machine file others can write.
	if err := fsutil.WriteFileAtomic(p, body, 0o600); err != nil {
		a.refuse("could not write ~/.abcd/oracle-routing.json (" + errText(err) + "); the routing was not accepted.")
		return
	}
	a.note(p)
}

// writeRepoRouting writes the repository table inside an os.Root at the
// checkout, so a symlinked .abcd/config a hostile repository committed cannot
// aim the write outside it.
func (a *applyCtx) writeRepoRouting(body []byte) {
	rel := layered.OracleRouting.RepoRel
	root, err := os.OpenRoot(a.cwd)
	if err != nil {
		a.refuse("could not open the repository to write " + rel + " (" + errText(err) + "); nothing was written.")
		return
	}
	defer root.Close()
	switch _, err := root.Lstat(filepath.FromSlash(rel)); {
	case err == nil:
		a.refuse("the repository's model-tier routing was not written: " + rel + " appeared while the question was open, and it is left as it is.")
		return
	case !errors.Is(err, fs.ErrNotExist):
		// The root refuses a path through a symlink that leaves the checkout;
		// name the symlink, which is the fault, rather than the root's error.
		if link := symlinkOnPath(root, rel); link != "" {
			a.refuse("the repository's model-tier routing was not written: " + link + " is a symlink, and " + rel +
				" is written only inside the repository, never through a link that leaves it.")
			return
		}
		a.refuse("the repository's model-tier routing was not written: " + rel + " could not be checked (" + errText(err) + "); nothing was written.")
		return
	}
	if err := fsutil.WriteFileAtomicInRoot(root, rel, body, 0o644); err != nil {
		a.refuse("could not write " + rel + " (" + errText(err) + "); the repository's routing was not accepted.")
		return
	}
	a.note(filepath.Join(a.cwd, filepath.FromSlash(rel)))
}

// proposalTable renders the bundled proposal in the routing file's own shape,
// one row per agent with its tier and fan-out bound.
func proposalTable() ([]byte, error) {
	type row struct {
		Tier   oracle.Tier `json:"tier"`
		FanOut int         `json:"fan_out"`
	}
	agents := map[string]row{}
	for agent, r := range oracle.Proposal() {
		agents[agent] = row{Tier: r.Tier, FanOut: r.FanOut}
	}
	doc := struct {
		SchemaVersion int            `json:"schema_version"`
		Agents        map[string]row `json:"agents"`
	}{layered.OracleRouting.SchemaVersion, agents}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// machineRoutingQuestion is the reason, the table and the question: core never
// prints, so the proposal travels as the text of the confirm.
func machineRoutingQuestion() string {
	var b strings.Builder
	b.WriteString("abcd proposes a model tier and a fan-out bound for each of its agents: frontier for the verdicts " +
		"a person reads and acts on, economy for the rest. Nothing of it applies until it is accepted. Accepting " +
		"writes this table to ~/.abcd/oracle-routing.json, where any row can be edited; a repository's own table " +
		"wins over it, and a step no configured provider can serve still runs through the harness, which is asked " +
		"for the tier. Declining writes nothing.\n")
	width := len("agent")
	for _, agent := range oracle.Roster() {
		width = max(width, len(agent))
	}
	fmt.Fprintf(&b, "  %-*s  %-12s  %s\n", width, "agent", "tier", "fan-out")
	proposal := oracle.Proposal()
	for _, agent := range oracle.Roster() {
		r := proposal[agent]
		fmt.Fprintf(&b, "  %-*s  %-12s  %d\n", width, agent, r.Tier, r.FanOut)
	}
	b.WriteString(oracleRoutingMachineQuestionTail)
	return b.String()
}

// repoRoutingQuestion is the separate repository offer.
func repoRoutingQuestion() string {
	return "The same table can also be committed with this repository, where it routes the delegated steps of " +
		"everyone who works in it and wins over each machine's own table. Declining writes nothing. " +
		oracleRoutingRepoQuestionTail
}

// symlinkOnPath names the first directory on rel's path, inside root, that is
// a symlink, or "" when none is.
func symlinkOnPath(root *os.Root, rel string) string {
	parts := strings.Split(rel, "/")
	for i := 1; i < len(parts); i++ {
		dir := strings.Join(parts[:i], "/")
		fi, err := root.Lstat(filepath.FromSlash(dir))
		if err != nil {
			return ""
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return dir
		}
	}
	return ""
}
