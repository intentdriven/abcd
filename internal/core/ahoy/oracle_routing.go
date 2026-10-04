package ahoy

// The oracle-routing consent `ahoy install` asks for (itd-2609170822093401
// AC 2, spc-2609180535002478 step 5). abcd ships its own proposal for the model
// tier and fan-out bound each agent deserves (internal/core/oracle), and
// nothing of it applies until it is accepted. The install step is where it is
// accepted:
//
//   - the machine offer says the proposal in counts (how many agents at each
//     tier, and their fan-out bounds) and, on consent, writes the full table
//     to ~/.abcd/oracle-routing.json, owner-only, because the resolver
//     refuses a machine file others can write;
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
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
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
	return abcdhome.Path(home, layered.OracleRouting.MachineRel)
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
			FixHint:  "ahoy install renders the proposal and writes " + abcdhome.Display("oracle-routing.json") + " only on consent; --yes never accepts it (run without --yes).",
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
		a.refuse("the model-tier routing was not written: the home directory could not be resolved, so " + abcdhome.Display("oracle-routing.json") + " has nowhere to go.")
		return
	}
	// The resolver refuses a machine table behind a symlinked ~/.abcd, so a
	// write through the link would land wherever it points and never be read.
	if err := fsutil.HomeScopeLink(userHome(), abcdhome.Rel(layered.OracleRouting.MachineRel)); err != nil {
		a.refuse("the model-tier routing was not written: " + err.Error() + ".")
		return
	}
	if !absent(p) {
		a.refuse("the model-tier routing was not written: " + abcdhome.Display("oracle-routing.json") + " appeared while the question was open, and it is left as it is.")
		return
	}
	// ~/.abcd is created, judged and opened relative to home's descriptor and
	// the table is written through it, so a link swapped in after the check
	// above is refused rather than written through (iss-2609281310017733).
	dir, err := fsutil.EnsureHomeScope(userHome(), abcdhome.Rel(), 0o700)
	if errors.Is(err, fsutil.ErrHomeScopeSymlinked) {
		a.refuse("the model-tier routing was not written: " + err.Error() + ".")
		return
	}
	if err != nil {
		a.refuse("could not create " + abcdhome.Display() + " for the model-tier routing (" + errText(err) + "); nothing was written.")
		return
	}
	defer dir.Close()
	if _, err := dir.Lstat(layered.OracleRouting.MachineRel); !errors.Is(err, os.ErrNotExist) {
		a.refuse("the model-tier routing was not written: " + abcdhome.Display("oracle-routing.json") + " appeared while the question was open, and it is left as it is.")
		return
	}
	// 0600, never wider: the resolver refuses a machine file others can write.
	if err := fsutil.WriteFileAtomicInRoot(dir, layered.OracleRouting.MachineRel, body, 0o600); err != nil {
		a.refuse("could not write " + abcdhome.Display("oracle-routing.json") + " (" + errText(err) + "); the routing was not accepted.")
		return
	}
	a.note(writeRouting, p)
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
	a.note(writeRouting, filepath.Join(a.cwd, filepath.FromSlash(rel)))
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

// routingTierReason says why the proposal puts an agent at a tier, for every
// tier the bundled proposal uses; a test fails on a proposed tier without one.
var routingTierReason = map[oracle.Tier]string{
	oracle.Frontier: "for verdicts a person reads and acts on",
	oracle.Economy:  "for the rest",
}

// proposalCounts says the proposal in counts, never naming an agent: how many
// agents, how many at each tier (in the order of oracle.Tiers() reversed:
// frontier before economy today, though a host-decides row would lead, which
// is not a strength order) with the reason that tier is proposed, and the fan-out
// bounds, one figure when every agent shares it and "<bound> for <number of
// agents>" pairs when they differ. A row per agent does not fit
// one question, so the product thinker ruled for counts on 2026-10-03
// (iss-2610031236155833); the written table still has every row.
func proposalCounts(roster []string, p oracle.Table) string {
	perTier := map[oracle.Tier]int{}
	perFanOut := map[int]int{}
	for _, agent := range roster {
		r := p[agent]
		perTier[r.Tier]++
		perFanOut[r.FanOut]++
	}
	var tiers []string
	vocab := oracle.Tiers()
	for i := len(vocab) - 1; i >= 0; i-- {
		n := perTier[vocab[i]]
		if n == 0 {
			continue
		}
		part := fmt.Sprintf("%d %s", n, vocab[i])
		if why := routingTierReason[vocab[i]]; why != "" {
			part += ", " + why
		}
		tiers = append(tiers, part)
	}
	bounds := make([]int, 0, len(perFanOut))
	for b := range perFanOut {
		bounds = append(bounds, b)
	}
	sort.Ints(bounds)
	var fanOut string
	switch len(bounds) {
	case 0:
		fanOut = "no fan-out"
	case 1:
		fanOut = fmt.Sprintf("fan-out %d each", bounds[0])
	default:
		pairs := make([]string, len(bounds))
		for i, b := range bounds {
			pairs[i] = fmt.Sprintf("%d for %d", b, perFanOut[b])
		}
		fanOut = "fan-out " + strings.Join(pairs, ", ")
	}
	return fmt.Sprintf("%d agents: %s; %s", len(roster), strings.Join(tiers, "; "), fanOut)
}

// machineRoutingQuestion is the reason, the proposal in counts and the
// question: core never prints, so the proposal travels as the text of the
// confirm. It says where every row can be read once accepted, since the
// question names no agent (proposalCounts).
func machineRoutingQuestion() string {
	return "abcd proposes a model tier and fan-out bound for its " + proposalCounts(oracle.Roster(), oracle.Proposal()) +
		". Accepting writes every row to " + abcdhome.Display("oracle-routing.json") + "; nothing applies before. " +
		"A repository's table wins; a step no provider serves goes to the harness, asked for its tier. " +
		"Declining writes nothing.\n" + oracleRoutingMachineQuestionTail
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
