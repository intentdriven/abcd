package capture

// itemfate.go is the ledger's answer to two questions the comparative channel
// asks (spc-2609020626039834; adr-2609021016272867).
//
// ItemFate: has anything happened to this candidate since the widening reading
// returned it? The candidate set the design fixes is the widening run's output
// BEFORE admission, so a candidate carrying a standing disposition or an
// admission is not one, and the comparative assembly refuses rather than handing
// a reading the researcher's judgement dressed as cold text.
//
// ComparativeRunFor: which comparative run characterised this widening run? It
// is the probe spc-2609020626040342's disposition gate reads — no disposition at
// the widening position until a committed comparative run names the item's run —
// and it lives here because `capture` does not import `reading` and therefore
// can be read from the disposition writer without a cycle.
//
// Both are WALKS over the ledger and the durable readings family. The JUDGEMENT
// is issueschema's (JudgeItemFate), on the rule that file states: two readers of
// one question are tolerable, two answers are not.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// admissionFileRe is the admission filename grammar, taken from the RESOLVER so
// this walk and the gate that judges committed admissions agree about which
// files are admissions. A stricter copy here would report a proposal as
// unadmitted that the gate has just accepted.
var admissionFileRe = recordid.FilenameNumRe(issueschema.AdmissionFamily)

// ItemFate reports what the ledger says has happened to one reading item of one
// run: the dispositions standing over it, and the admissions naming it as their
// proposal.
//
// The admission side is keyed on the (run, proposal) PAIR, exactly as
// core/lint's admittedProposals keys it, and for the reason that function
// records: keying on the proposal alone made a proposal id a global silencer
// (iss-2608300935215868). An admission whose own `run` field disagrees with the
// directory it sits in counts under neither — the record contradicts itself
// about which candidate set it joined, and honouring either half would let the
// other lie.
//
// An absent store is a STATE, not a fault: a repository that has dispositioned
// or admitted nothing has an item with no fate, which is exactly what a
// comparative assembly needs to find.
func ItemFate(repoRoot, run, item string) (issueschema.ItemFate, error) {
	if !recordid.ValidReadingRunID(run) {
		return issueschema.ItemFate{}, fmt.Errorf("%w: run %q does not match ^%s-[0-9]+$",
			ErrMalformedFrontmatter, run, issueschema.ReadingRunFamily)
	}
	if !recordid.ValidReadingItemID(item) {
		return issueschema.ItemFate{}, fmt.Errorf("%w: item %q does not match ^%s-[0-9]+$",
			ErrMalformedFrontmatter, item, issueschema.ReadingItemFamily)
	}
	issuesRoot := filepath.Join(repoRoot, filepath.FromSlash(LedgerRelPath))

	itemDir := filepath.Join(issuesRoot, issueschema.DispositionsDir, item)
	records, err := readDispositions(itemDir)
	if err != nil {
		return issueschema.ItemFate{}, err
	}
	admissions, err := admissionsNaming(issuesRoot, run, item)
	if err != nil {
		return issueschema.ItemFate{}, err
	}
	return issueschema.JudgeItemFate(records, admissions), nil
}

// admissionsNaming lists the ids of the admissions filed under run that name
// item as their proposal.
func admissionsNaming(issuesRoot, run, item string) ([]string, error) {
	runDir := filepath.Join(issuesRoot, issueschema.AdmissionsDir, run)
	// The family root as well as the leaf: guarding only the leaf leaves the
	// answer coming from outside the ledger through a symlinked admissions/.
	if err := refuseSymlinkedDir(filepath.Dir(runDir)); err != nil {
		return nil, err
	}
	if err := refuseSymlinkedDir(runDir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(runDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !admissionFileRe.MatchString(e.Name()) {
			continue
		}
		content, err := readRecordGuarded(filepath.Join(runDir, e.Name()))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			// An admission the walk cannot read supports no claim that the item is
			// UNADMITTED, and this probe's whole use is to prove an item free. So
			// the read failure is returned rather than skipped: the comparative
			// assembly refuses on it, which is the fail-closed direction.
			return nil, fmt.Errorf("%w: reading the admission %s of run %s: %v",
				ErrPathUnsafe, e.Name(), run, err)
		}
		fm, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return nil, fmt.Errorf("%w: the admission %s of run %s does not parse: %v",
				ErrMalformedFrontmatter, e.Name(), run, err)
		}
		if asString(fm["run"]) != run || asString(fm["proposal"]) != item {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), ".md"))
	}
	sort.Strings(out)
	return out, nil
}

// ComparativeRunFor returns the lowest committed comparative run id whose
// `candidate_run` names run, or the empty string when none does.
//
// It reads the RUN RECORDS under the durable readings family, which is where a
// run's commit marker lives: a directory holding a parked manifest and no run
// record is a run that never happened, and it answers nothing here.
//
// The LOWEST match rather than any match, so two comparative runs over one
// widening run — a legitimate state, since a second comparative run before any
// disposition is a second run — resolve to one answer that does not depend on
// the order the filesystem returned.
func ComparativeRunFor(repoRoot, run string) (string, error) {
	if !recordid.ValidReadingRunID(run) {
		return "", fmt.Errorf("%w: run %q does not match ^%s-[0-9]+$",
			ErrMalformedFrontmatter, run, issueschema.ReadingRunFamily)
	}
	runsRoot := filepath.Join(repoRoot, filepath.FromSlash(issueschema.ReadingsRecordDir))
	if err := refuseSymlinkedDir(runsRoot); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(runsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() || !recordid.ValidReadingRunID(e.Name()) {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		dir := filepath.Join(runsRoot, name)
		if err := refuseSymlinkedDir(dir); err != nil {
			return "", err
		}
		head, ok, err := readRunHead(filepath.Join(dir, issueschema.RunRecordFileName))
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		if head.Position == PositionComparative && head.CandidateRun == run {
			return name, nil
		}
	}
	return "", nil
}

// PositionComparative is the comparative position's token, as the run record
// spells it. It is stated here rather than imported because core/reading imports
// this package and not the other way round; issueschema.ReadingPositions is the
// vocabulary both are held to.
const PositionComparative = "comparative"

// readRunHead decodes one run record's candidate-join subset. A missing file is
// "not a committed run", not a fault: the marker's absence is the state.
func readRunHead(path string) (issueschema.RunHead, bool, error) {
	raw, err := fsutil.ReadGuarded(path, issueschema.RecordReadLimit)
	if err != nil {
		if os.IsNotExist(err) {
			return issueschema.RunHead{}, false, nil
		}
		return issueschema.RunHead{}, false, fmt.Errorf("%w: reading %s: %v", ErrPathUnsafe, path, err)
	}
	var head issueschema.RunHead
	if err := json.Unmarshal(raw, &head); err != nil {
		return issueschema.RunHead{}, false, fmt.Errorf("%w: %s does not decode as a run record: %v",
			ErrMalformedFrontmatter, path, err)
	}
	return head, true, nil
}
