package reading

// candidates.go derives the widening run a comparative reading characterises,
// and holds the two refusals that derivation carries (adr-2609021016272867;
// spc-2609020626039834, "The derived run" and "The ordering guard").
//
// No operand names the run. The design fixes the invocation at a position and a
// target state (framework v4 section 8.2 and ruling M8; companion v4 section
// 4.1), brief invariant 15's operand enumeration names two, and
// adr-2609021016286571 restores that letter — so the run comes from the record.
// The rule is the ADR's: the one committed widening run at the target whose
// items carry no disposition and no admission. None, or more than one, refuses
// and lists the widening runs at the target, because the operator's next act —
// dispositioning one run's items — is what the design sequences after the
// comparative reading anyway, and they can only perform it if they are told what
// there is to disposition.
//
// # How "at the target" is read here, and the ruling that is owed
//
// The ADR's phrase is "the committed widening run AT THE TARGET", and read as
// commit equality alone it makes the loop the design sequences unrunnable.
// `reading ingest` writes a widening run's reading records into the committed
// ledger, where they are uncommitted by construction; the comparative position's
// candidate row reaches that store, so the next assembly refuses on the dirty
// gate naming those very records. Committing them — the act the design places
// between the ingest and the next reading — moves HEAD, and a run whose recorded
// target must EQUAL the assembly's target no longer has one
// (iss-2609021833302981).
//
// So the phrase is read as reaching an ancestor whose OBJECT SET has not moved.
// A committed widening run qualifies when its recorded target equals the target,
// or when its recorded target is an ancestor of the target and every path
// changed between the two lies inside the readings store and the issue ledger's
// own record families. A run whose target is not an ancestor is not a run at
// this target and is not listed; a run where anything else changed is listed and
// refused, naming the first such path, because the object set it read is not the
// object set this assembly names.
//
// The ground is the divergence register's entry 27 — "the object set, not the
// commit, names what the readings are about" — and the companion's section 7.2
// with ruling R4, which fix the comparative reading's object as the widening
// run's returned items and the criteria discipline and nothing from the tree.
// The commit the run names is how the object set is identified, not what the
// reading was about, so a commit that moved only the instrument's own record
// leaves the object set where it was.
//
// This is an INTERPRETATION and the maintainer's ruling is owed
// (iss-2609021857343626). The alternative it was chosen over is an assembly
// whose target may differ from HEAD at this position — legitimate, because the
// comparative position reads no working tree — and the register gains the entry
// either way.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// WideningRun is one widening run at a target, as the derivation sees it: what
// it is, how much it holds, and whether anything has happened to what it holds.
//
// Every field is on the LISTING a refusal renders, because the operator's next
// move depends on all of them: a run that did not qualify is shown rather than
// hidden, with the reason it did not.
type WideningRun struct {
	// ID is the run identifier (rdg-N).
	ID string `json:"id"`
	// Target is the commit the run's own record names: the commit its reading
	// actually read. It is the assembly's target or an ancestor of it, and the
	// manifest carries both so a reader can diff them.
	Target string `json:"target"`
	// Items is how many reading records the run holds in the ledger.
	Items int `json:"items"`
	// Dispositioned reports that at least one item carries a standing
	// disposition; Admitted, that at least one carries an admission. Either is a
	// fate, and a candidate set is defined as pre-admission.
	Dispositioned bool `json:"dispositioned"`
	Admitted      bool `json:"admitted"`
	// Committed reports that the run reached its commit marker
	// (`run.json`). A run without one never happened and its records are the
	// next ingest sweep's to roll back, so it is listed and never selected.
	Committed bool `json:"committed"`
	// Fated names what stands over the items that carry a fate, one rendered
	// phrase per item, so a refusal names THE ITEM AND THE DISPOSITION rather
	// than only the run. The two booleans above answer "did this run qualify";
	// this answers "which item stopped it", which is what the ordering guard
	// owes an operator (spc-2609020626039834, "The ordering guard").
	Fated []string `json:"fated,omitempty"`
	// Moved names the FIRST path that changed between this run's own target and
	// the assembly's, outside the readings store and the issue ledger's record
	// families. It is empty when the two targets are the same commit, and empty
	// when everything that changed between them was the instrument's own record.
	//
	// A run carrying it is LISTED and never selected: its items characterise an
	// object set this target no longer holds, and the operator needs the path to
	// see that (the reading of "at the target" this file's head comment states).
	Moved string `json:"object_set_moved,omitempty"`
}

// Qualifies reports whether this run is the candidate set the ADR describes: a
// committed widening run holding items, none of which carries a fate, read over
// the object set this assembly's target still holds.
func (r WideningRun) Qualifies() bool {
	return r.Committed && r.Items > 0 && !r.Dispositioned && !r.Admitted &&
		len(r.Fated) == 0 && r.Moved == ""
}

// Line renders one run for the refusal listing: one run per line, saying what it
// holds and why it did or did not qualify.
func (r WideningRun) Line() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %d item(s)", r.ID, r.Items)
	if !r.Committed {
		b.WriteString(", NOT committed (no ")
		b.WriteString(issueschema.RunRecordFileName)
		b.WriteString(" commit marker, so the run never happened)")
	}
	switch {
	case len(r.Fated) > 0:
		b.WriteString(", ")
		b.WriteString(strings.Join(r.Fated, "; "))
	case r.Items == 0:
		b.WriteString(", nothing to characterise")
	case r.Committed:
		b.WriteString(", no disposition and no admission")
	}
	if r.Moved != "" {
		fmt.Fprintf(&b, "; the object set changed since the run: %s changed between %s, which "+
			"this run read, and the target", r.Moved, shortTarget(r.Target))
	}
	return b.String()
}

// renderRuns renders the whole listing, one run per line, or says there is none.
func renderRuns(runs []WideningRun) string {
	if len(runs) == 0 {
		return "  (no widening run is committed at this target)"
	}
	lines := make([]string, 0, len(runs))
	for _, r := range runs {
		lines = append(lines, "  "+r.Line())
	}
	return strings.Join(lines, "\n")
}

// NoCandidateRun is the refusal when no widening run at the target qualifies. It
// carries the whole listing, because the remedy depends on which of the reasons
// applies: no run at all, a run that never committed, a run holding nothing, or
// a run whose items already carry a fate.
type NoCandidateRun struct {
	Target string
	Runs   []WideningRun
}

func (e *NoCandidateRun) Error() string {
	return fmt.Sprintf("the %s assembly derives its candidate set from the record, and no committed "+
		"widening run at %s qualifies, so there is nothing to characterise. A run qualifies when "+
		"every item it holds is free of a disposition and an admission — the candidate set is "+
		"defined as PRE-ADMISSION: characterisation precedes admission, and a candidate whose "+
		"fate is already recorded is not one — and when nothing outside the readings store and "+
		"the issue ledger changed between the commit the run read and this target "+
		"(adr-2609021016272867). The widening runs at this target:\n%s",
		PositionComparative, shortTarget(e.Target), renderRuns(e.Runs))
}

// AmbiguousCandidateRun is the refusal when more than one qualifies. The remedy
// is stated because the design already sequences it: dispositioning one run's
// items is the act that follows a comparative reading, and performing it makes
// the selection unambiguous.
type AmbiguousCandidateRun struct {
	Target string
	Runs   []WideningRun
}

func (e *AmbiguousCandidateRun) Error() string {
	qualifying := make([]string, 0, len(e.Runs))
	for _, r := range e.Runs {
		if r.Qualifies() {
			qualifying = append(qualifying, r.ID)
		}
	}
	return fmt.Sprintf("the %s assembly derives its candidate set from the record, and %d widening "+
		"runs at %s have every item free of a disposition and an admission (%s); the invocation is "+
		"a position and a target state, so nothing names which. Disposition one run's items — the "+
		"act the design places after the comparative reading in any case — and the selection is "+
		"unambiguous. The widening runs at this target:\n%s",
		PositionComparative, len(qualifying), shortTarget(e.Target),
		strings.Join(qualifying, ", "), renderRuns(e.Runs))
}

// shortTarget abbreviates a commit for a refusal message.
func shortTarget(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// WideningRuns lists every widening run at the target, committed or not, with
// its item count and the fate of its items.
//
// It reads the DURABLE run family for what a run is — its position, its target,
// and whether it committed — and the LEDGER for what it holds. Those are the two
// places the two facts actually live: a run's identity is in
// `.abcd/development/readings/<run>/`, and its items are reading records under
// the issue ledger.
//
// A directory carrying neither a run record nor a manifest is not a run and is
// skipped: it says nothing about a position or a target, so listing it would
// name a run this function cannot describe.
func WideningRuns(repoRoot, target string) ([]WideningRun, error) {
	runsRoot := filepath.Join(repoRoot, filepath.FromSlash(issueschema.ReadingsRecordDir))
	if err := refuseSymlinkedRunDir(runsRoot); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(runsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading: listing the committed runs: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() || !recordid.ValidReadingRunID(e.Name()) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	var out []WideningRun
	for _, name := range names {
		dir := filepath.Join(runsRoot, name)
		if err := refuseSymlinkedRunDir(dir); err != nil {
			return nil, err
		}
		head, committed, err := runHeadOf(dir)
		if err != nil {
			return nil, err
		}
		if head.position != string(PositionWidening) {
			continue
		}
		reaches, moved, err := reachesTarget(repoRoot, head.target, target)
		if err != nil {
			return nil, err
		}
		if !reaches {
			continue
		}
		run := WideningRun{ID: name, Target: head.target, Committed: committed, Moved: moved}
		items, err := ledgerItemsOf(repoRoot, name)
		if err != nil {
			return nil, err
		}
		run.Items = len(items)
		for _, item := range items {
			fate, err := capture.ItemFate(repoRoot, name, item)
			if err != nil {
				return nil, fmt.Errorf("reading: reading the fate of %s in run %s: %w", item, name, err)
			}
			switch {
			case fate.Cyclic:
				run.Dispositioned = true
				run.Fated = append(run.Fated, fmt.Sprintf(
					"%s carries a supersession cycle, so its fate cannot be read and it is not "+
						"demonstrably uncommitted", item))
			case fate.Contested():
				run.Dispositioned = true
				run.Fated = append(run.Fated, fmt.Sprintf(
					"%s carries %d standing dispositions (%s), so no reader may say which is in force",
					item, len(fate.Dispositions), strings.Join(fate.Dispositions, ", ")))
			case len(fate.Dispositions) > 0:
				run.Dispositioned = true
				run.Fated = append(run.Fated, fmt.Sprintf(
					"%s carries the standing disposition %s", item, strings.Join(fate.Dispositions, ", ")))
			}
			if len(fate.Admissions) > 0 {
				run.Admitted = true
				run.Fated = append(run.Fated, fmt.Sprintf(
					"%s carries the admission %s", item, strings.Join(fate.Admissions, ", ")))
			}
		}
		out = append(out, run)
	}
	return out, nil
}

// reachesTarget answers, for one run's recorded target, whether that run is a
// run AT the assembly's target — and, when the two commits differ, names the
// first path whose change says the object set moved between them.
//
// The three outcomes are kept apart because the operator needs them apart. A run
// at the target itself, or at an ancestor of it that only the instrument's own
// record separates from it, REACHES and has moved nothing: it is a candidate.
// A run at an ancestor across which something else changed REACHES and names
// what changed: it is listed and refused, so the operator sees why. A run whose
// target is not an ancestor at all does not reach: it is not a run at this
// target, no diff between the two says anything, and listing it would name a run
// this target has nothing to do with.
//
// A recorded target that does not resolve in this repository is treated as not
// reaching rather than as a failure. It is a run record naming a commit this
// checkout cannot see — carried in from elsewhere, or left by a rewritten
// history — and the honest thing to say about it is that it is not a run at this
// target, not to fail every derivation the repository can perform.
func reachesTarget(repoRoot, runTarget, target string) (bool, string, error) {
	if runTarget == "" {
		return false, "", nil
	}
	if runTarget == target {
		return true, "", nil
	}
	if _, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "--quiet", runTarget+"^{commit}"); err != nil {
		return false, "", nil
	}
	ancestor, err := gitutil.IsAncestor(repoRoot, runTarget, target)
	if err != nil {
		return false, "", fmt.Errorf("reading: comparing the widening run's target %s with %s: %w",
			shortTarget(runTarget), shortTarget(target), err)
	}
	if !ancestor {
		return false, "", nil
	}
	// --no-renames, because rename detection reports only the destination of a
	// rename: a source file moved INTO the ledger would then read as a ledger-only
	// change. Both halves must be named for the check to be the check it claims.
	// -z, because a path is not escaped or quoted in that form, so core.quotepath
	// cannot change the format under the parser.
	out, err := gitutil.RunCapped(repoRoot, 8<<20, "diff", "--name-only", "--no-renames", "-z",
		runTarget, target)
	if err != nil {
		return false, "", fmt.Errorf("reading: listing what changed between the widening run's "+
			"target %s and %s: %w", shortTarget(runTarget), shortTarget(target), err)
	}
	var moved []string
	for _, p := range strings.Split(out, "\x00") {
		if p == "" || instrumentRecordPath(p) {
			continue
		}
		moved = append(moved, p)
	}
	if len(moved) == 0 {
		return true, "", nil
	}
	// Sorted, so the path a refusal names is the same path on every git and every
	// filesystem: "the first such path" has to be a fact about the diff rather
	// than about the order this run of git happened to emit it in.
	sort.Strings(moved)
	return true, moved[0], nil
}

// instrumentRecordPath reports whether a path is the instrument's own record
// rather than an object a reading is about: the durable readings family, and the
// issue ledger's own record families.
//
// The ledger list is issueschema.LedgerDirs() and never a hand-kept copy, for
// the reason that function states — a family added later must be covered the day
// its constant is declared, or this check would call a new family's records a
// moved object set.
func instrumentRecordPath(rel string) bool {
	if strings.HasPrefix(rel, issueschema.ReadingsRecordDir+"/") {
		return true
	}
	for _, dir := range issueschema.LedgerDirs() {
		if strings.HasPrefix(rel, capture.LedgerRelPath+"/"+dir+"/") {
			return true
		}
	}
	return false
}

// DeriveCandidateRun applies the ADR's rule over the runs at the target: exactly
// one must qualify.
//
// The two refusals carry the whole listing rather than only what went wrong,
// because the operator resolves either of them by looking at the runs: with none
// they need to see whether a run is uncommitted, empty or already dispositioned,
// and with more than one they need to see which to disposition.
func DeriveCandidateRun(repoRoot, target string) (WideningRun, error) {
	runs, err := WideningRuns(repoRoot, target)
	if err != nil {
		return WideningRun{}, err
	}
	var qualifying []WideningRun
	for _, r := range runs {
		if r.Qualifies() {
			qualifying = append(qualifying, r)
		}
	}
	switch len(qualifying) {
	case 1:
		return qualifying[0], nil
	case 0:
		return WideningRun{}, &NoCandidateRun{Target: target, Runs: runs}
	default:
		return WideningRun{}, &AmbiguousCandidateRun{Target: target, Runs: runs}
	}
}

// runHead is the subset of a run's own artefacts the derivation reads: which
// position it read at, and which commit it read.
type runHead struct {
	position string
	target   string
}

// runHeadOf reads one run directory's head, and reports whether the run
// COMMITTED. The run record is the commit marker and is read first; a directory
// without one is described from its parked manifest instead, so an uncommitted
// run is listed with its position and target rather than silently absent.
func runHeadOf(dir string) (runHead, bool, error) {
	var record struct {
		Position     string `json:"position"`
		TargetCommit string `json:"target_commit"`
	}
	err := readRunJSON(filepath.Join(dir, issueschema.RunRecordFileName), &record)
	switch {
	case err == nil:
		return runHead{position: record.Position, target: record.TargetCommit}, true, nil
	case !os.IsNotExist(err):
		return runHead{}, false, err
	}
	var parked struct {
		Position     string `json:"position"`
		TargetCommit string `json:"target_commit"`
	}
	if err := readRunJSON(filepath.Join(dir, ManifestFileName), &parked); err != nil {
		if os.IsNotExist(err) {
			return runHead{}, false, nil
		}
		return runHead{}, false, err
	}
	return runHead{position: parked.Position, target: parked.TargetCommit}, false, nil
}

// readRunJSON decodes one of a run's own artefacts, leaving os.ErrNotExist
// unwrapped so the caller can tell "absent" from "unreadable".
func readRunJSON(path string, into any) error {
	raw, err := fsutil.ReadGuarded(path, MaxFileBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return fmt.Errorf("reading: %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("reading: %s does not decode: %w", path, err)
	}
	return nil
}

// ledgerItemsOf lists the reading-record ids one run holds in the ledger, sorted.
// An absent directory is a run with no items, which is a state and not a fault.
func ledgerItemsOf(repoRoot, run string) ([]string, error) {
	dir := filepath.Join(repoRoot, filepath.FromSlash(CandidateSource), run)
	if err := refuseSymlinkedRunDir(filepath.Dir(dir)); err != nil {
		return nil, err
	}
	if err := refuseSymlinkedRunDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading: listing the items of run %s: %w", run, err)
	}
	var out []string
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".md")
		if e.IsDir() || !ok || !recordid.ValidReadingItemID(id) {
			continue
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// refuseSymlinkedRunDir refuses a path that exists and is not a real directory.
// It is the same stance core/capture takes on the same ledger tree: a run's
// directory is a run's directory, never a pointer at somebody else's — and here
// the consequence is what a reading is HANDED, so following one would let a link
// decide the candidate set.
func refuseSymlinkedRunDir(dir string) error {
	fi, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading: %s could not be examined: %w", dir, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
		return fmt.Errorf("reading: %s is not a real directory (a symlink is not followed); the "+
			"candidate set is decided by the record, never by a link", dir)
	}
	return nil
}

// refuseNonWideningCandidate is the record-level check on every file the
// candidate row enumerates: it must be a well-formed reading record, and it must
// have been returned at the WIDENING position.
//
// A run at any other position is not a candidate set — the design fixes the
// comparative object as the widening reading's returned configurations and
// admits no other source (companion 7.2, R4) — and the row's own path cannot
// establish that, because every position's records live in the same family.
func refuseNonWideningCandidate(rel string, raw []byte) error {
	fm, err := capture.ValidateReadingRecord(string(raw))
	if err != nil {
		return fmt.Errorf("reading: %s is admitted as a candidate and is not a well-formed reading "+
			"record: %w", rel, err)
	}
	if got, _ := fm["position"].(string); got != string(PositionWidening) {
		return fmt.Errorf("reading: %s was returned at the %s position, and a candidate set is one "+
			"WIDENING run's returned configurations; a run at any other position is not a candidate "+
			"set (adr-2609021016272867)", rel, got)
	}
	return nil
}

// assertCandidateProjection is the fail-closed half of the readings-store
// exclusion the manifest declares at the comparative position.
//
// The directory rows of the floor are enforced by path, and a path needs no
// parse; the readings row is a SIGNAL row instead, because a directory row there
// would be false — one run's items do travel. So what the manifest asserts is
// the narrower promise, and this is what makes it a promise rather than a
// disclosure: no candidate item may be emitted from outside the derived run's
// directory, or under a field outside the two projected, and nothing else may be
// emitted from the readings store at all (adr-56; brief invariant 16).
func assertCandidateProjection(cands []candidate, run string) error {
	prefix := CandidateSource + "/" + run + "/"
	fields := map[string]bool{}
	for _, f := range CandidateFields {
		fields[f] = true
	}
	for _, c := range cands {
		underStore := strings.HasPrefix(c.path, CandidateSource+"/")
		if c.kind != KindCandidate {
			if underStore {
				return fmt.Errorf("reading: item %s comes from the readings store as a %s item; the "+
					"only thing that store supplies is the derived run's candidates, which the "+
					"manifest asserts", c.path, c.kind)
			}
			continue
		}
		if !strings.HasPrefix(c.path, prefix) {
			return fmt.Errorf("reading: candidate item %s lies outside the derived run %s, whose "+
				"exclusion the manifest asserts: every run other than the candidate run stays behind",
				c.path, run)
		}
		if !fields[c.field] {
			return fmt.Errorf("reading: candidate item %s projects the field %q, and the manifest "+
				"asserts that every field of the named run's items other than %s stays behind",
				c.path, c.field, strings.Join(CandidateFields, " and "))
		}
	}
	return nil
}

// candidateItems reports how many DISTINCT reading records the collected
// candidates came from, so the count the manifest records as `candidates` can be
// checked against what the row actually enumerated.
func candidateItems(cands []candidate) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cands {
		if c.kind != KindCandidate || seen[c.path] {
			continue
		}
		seen[c.path] = true
		out = append(out, candidateIDOf(c.path))
	}
	sort.Strings(out)
	return out
}

// candidateIDOf is the item id a candidate's path names. The record's file is
// named for its id, which is what lets the bundle tell a reading which rdi-N
// each text belongs to without carrying the path (brief invariant 15).
func candidateIDOf(rel string) string {
	return strings.TrimSuffix(filepath.Base(rel), ".md")
}
