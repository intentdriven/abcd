package reading

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/lint"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// AssembleRequest is one assembly: a position, a target commit, and where the
// two artefacts go. It carries no free-text operand of any kind, because there
// is no channel through which ledger content may travel in the framing of a
// request (ruling (5)).
type AssembleRequest struct {
	// RepoRoot is the repository the assembly reads.
	RepoRoot string
	// Position is the reading position. The set is closed.
	Position Position
	// Target is "HEAD" or a hexadecimal commit sha of 7 to 40 digits. Branch
	// names and tags are refused as mutable: the manifest's re-runnability rests
	// on a reference that cannot move.
	Target string
	// OutDir is the operator-named directory the two artefacts are written to.
	// Empty means the default local-tier run directory.
	OutDir string
	// OutDirLabel is how the OPERATOR spelled OutDir, used in refusal messages so
	// a path they did not type is never quoted back at them. The front door
	// resolves a relative --out against the working directory before calling in,
	// and scrubPaths cannot redact the result when the working directory is not a
	// prefix of it. Empty means OutDir is the operator's own spelling.
	OutDirLabel string
	// DryRun writes nothing into the repository's own tiers. With OutDir set the
	// artefacts still land there; with OutDir empty nothing is written at all
	// and the result is rendered only.
	DryRun bool
}

// AssembleResult is what one assembly produced. The bundle and the manifest are
// carried for a caller that wants them in memory, and the artefacts are named by
// their basenames rather than by a full path.
//
// OutDir is whatever the CALLER passed in, echoed back so it can find what it
// asked for. The core takes a relative one against the repository root, which
// suits the default it computes itself; the CLI resolves an operator's relative
// --out against the working directory before calling in, and then puts the
// operator's own string back on the result, so no absolute path nobody typed
// reaches the success surface. Neither ARTEFACT carries an output path at all.
type AssembleResult struct {
	RunID            string        `json:"run_id"`
	Position         Position      `json:"position"`
	TargetCommit     string        `json:"target_commit"`
	AssemblerVersion string        `json:"assembler_version"`
	ItemCount        int           `json:"item_count"`
	ManifestHash     string        `json:"manifest_hash"`
	Preset           AppliedPreset `json:"preset"`
	Size             SizeReport    `json:"size"`
	// CandidateRun is the widening run this assembly derived, and Candidates the
	// count of items it holds. Both are empty at every position but comparative,
	// where the operator has to be told which run the reading is about — no
	// operand named it, so the result is where they learn it
	// (adr-2609021016272867).
	CandidateRun string `json:"candidate_run,omitempty"`
	Candidates   int    `json:"candidates,omitempty"`
	// NotExercised reports the fixed interpretation: the derived run held fewer
	// than two candidates, so the comparative reading has nothing to compare and
	// the position was not exercised. The assembly still stages a run, and
	// ingesting it commits a comparative run with an empty item set naming that
	// widening run (the 2026-09-02 interpretations ruling; companion 7.6).
	NotExercised bool     `json:"not_exercised,omitempty"`
	OutDir       string   `json:"out_dir,omitempty"`
	Artefacts    []string `json:"artefacts"`
	Written      bool     `json:"written"`

	Bundle   Bundle   `json:"-"`
	Manifest Manifest `json:"-"`
}

// tokenBytesPerToken is the divisor of the byte-derived token estimate,
// measured rather than assumed: every tracked file in this repository was
// tokenized through tiktoken under cl100k_base and o200k_base (which agreed
// within 0.3 per cent), giving a byte-weighted 3.865 bytes per token across
// 17,119,789 bytes. 3.85 is that figure rounded.
//
// It is a proxy. tiktoken is OpenAI's tokenizer and no reader's actual
// tokenization is measured here or claimed, which is why every surface labels
// the figure an estimate. A single divisor mis-states each kind in known
// directions — test by about -7.7 per cent, prose by about +7 per cent — and
// that bias is disclosed in spc-68 rather than removed, because the estimate
// exists to judge plausibility at the order of magnitude. If it ever changes a
// decision it should not have it is replaced by a real tokenizer, never tuned
// (itd-198).
const tokenBytesPerToken = 3.85

// KindSize is one material kind's contribution to an assembly's weight.
type KindSize struct {
	Kind      Kind `json:"kind"`
	Items     int  `json:"items"`
	Bytes     int  `json:"bytes"`
	TokensEst int  `json:"tokens_est"`
}

// SizeReport is what an assembly would cost a reader, per material kind and in
// total. It rides on the result and never on the bundle: a reading has no use
// for its own weight, and itd-198 ac-8 holds the bundle's shape unchanged.
//
// Bytes count the item text that actually travels, not the file on disk, so a
// projected record is counted at what the reading receives rather than at what
// the record weighs. No budget is enforced and none is invented: the assembler
// cannot know what a given reader accepts.
type SizeReport struct {
	ByKind []KindSize `json:"by_kind"`
	Items  int        `json:"items"`
	// Unscanned is how many of those items the exclusion floor never examined,
	// which is the operator's own figure for how much of the assembly travels
	// on disclosure rather than on a scan. It rides on the result like the rest
	// of this report and on neither artefact, so it moves no version: the
	// per-item truth is the manifest's `scan` mark, and this is its total
	// (itd-194).
	Unscanned int    `json:"unscanned"`
	Bytes     int    `json:"bytes"`
	TokensEst int    `json:"tokens_est"`
	Basis     string `json:"basis"`
	// Window is the declaration the committed entry for this position carries,
	// nil where none is declared — a version 1 preset file. It rides on the
	// report rather than on either artefact, so a declaration is a fact about
	// the run's cost and never a field a reading or an auditor reads
	// (spc-2609020626048722).
	Window *Window `json:"window,omitempty"`
	// ExceedsWindow is whether this assembly measured past that declaration.
	// Nothing is refused for it: the eval is what fails, and the operator is
	// told here.
	ExceedsWindow bool `json:"exceeds_window"`
	// OverTarget is whether the total estimate exceeds TargetTokens. The target
	// is a statement to the operator and never a refusal (maintainer's ruling of
	// 2026-09-02; divergence register 24).
	OverTarget bool `json:"over_target"`
	// Mechanism is the entailment position's yield bound: how many projected
	// intents carry a mechanism claim. It is nil at every other position,
	// because the readings companion's section 6.6 asks for it beside the
	// entailment reading's findings and nowhere else.
	Mechanism *MechanismReport `json:"mechanism,omitempty"`
}

// MechanismReport is the proportion the readings companion's section 6.6 asks
// be reported beside the entailment reading's findings (divergence register 16;
// iss-2609012259585189).
//
// The count is per FILE, derived from the candidates by path and field, so it is
// checkable against the manifest, which already records which fields each file
// contributed. Nothing is added to the manifest's shape for it.
//
// The three lower counts are exhaustive over Intents: a projected intent file
// either contributed a Mechanism item whose text is the nullity, one whose text
// is anything else, or no Mechanism item at all. The workstream's own shipped
// intents keep their absent claims as the Iteration 1 baseline and are reported,
// never backfilled (maintainer's ruling of 2026-09-02).
type MechanismReport struct {
	Intents    int `json:"intents"`
	Stated     int `json:"stated"`
	NoneStated int `json:"none_stated"`
	Absent     int `json:"absent"`
}

// mechanismField is the projected field a mechanism claim travels as, and
// mechanismNullity is the text a record writes when it has none. Both are the
// record's own spelling: the framework's section 7.1 fixes `## Mechanism` as the
// causal claim's home, and the nullity is what an intent carries in place of one.
const (
	mechanismField   = "Mechanism"
	mechanismNullity = "None stated."
)

// mechanismReport counts the projected intent files in one assembly by whether
// they carried a mechanism claim.
func mechanismReport(cands []candidate) *MechanismReport {
	// Two passes over paths rather than one, because a file's verdict depends on
	// whether ANY of its candidates is the Mechanism field, and the candidates
	// of one file are not guaranteed adjacent to a reader of this function.
	files := map[string]string{}
	seen := map[string]bool{}
	order := make([]string, 0, len(cands))
	for _, c := range cands {
		if c.kind != KindIntentProjection {
			continue
		}
		if !seen[c.path] {
			seen[c.path] = true
			order = append(order, c.path)
		}
		if c.field == mechanismField {
			files[c.path] = strings.TrimSpace(c.text)
		}
	}
	rep := &MechanismReport{Intents: len(order)}
	for _, p := range order {
		text, ok := files[p]
		switch {
		case !ok:
			rep.Absent++
		case text == mechanismNullity:
			rep.NoneStated++
		default:
			rep.Stated++
		}
	}
	return rep
}

// sizeBasis names the method and the divisor inside the artefact, so a report
// read out of context still says what it is rather than looking like a count.
// It carries no outer parentheses of its own: the CLI wraps it in a
// parenthetical, and a string that brought its own rendered as a doubled pair.
// The rehearsal caught that; no fixture could have.
var sizeBasis = fmt.Sprintf("estimated: bytes / %.2f, byte-derived, not a tokenizer's count", tokenBytesPerToken)

// sizeReport totals the collected candidates by kind, in the closed
// vocabulary's order. A kind that passed no item is omitted rather than
// reported as zero: an absent kind and an empty one are different facts, and
// the manifest can settle which.
func sizeReport(cands []candidate, position Position, window *Window) SizeReport {
	byKind := make(map[Kind]*KindSize, len(Kinds()))
	rep := SizeReport{Basis: sizeBasis}
	for _, c := range cands {
		k, ok := byKind[c.kind]
		if !ok {
			k = &KindSize{Kind: c.kind}
			byKind[c.kind] = k
		}
		n := len(c.text)
		k.Items++
		k.Bytes += n
		rep.Items++
		rep.Bytes += n
		if c.scan == ScanUnscanned {
			rep.Unscanned++
		}
	}
	rep.ByKind = make([]KindSize, 0, len(byKind))
	for _, kind := range Kinds() {
		k, ok := byKind[kind]
		if !ok {
			continue
		}
		k.TokensEst = estimateTokens(k.Bytes)
		rep.ByKind = append(rep.ByKind, *k)
	}
	rep.TokensEst = estimateTokens(rep.Bytes)
	rep.OverTarget = rep.TokensEst > TargetTokens
	rep.Window = window
	rep.ExceedsWindow = window != nil && rep.TokensEst > window.TokensEst
	if position == PositionEntailment {
		rep.Mechanism = mechanismReport(cands)
	}
	return rep
}

// estimateTokens converts bytes to the byte-derived token estimate.
func estimateTokens(b int) int {
	return int(float64(b) / tokenBytesPerToken)
}

// BundleFileName and ManifestFileName are the two artefacts an assembly writes:
// separate files, so the assembled input can go to a reader while the manifest
// stays with the auditor.
const (
	BundleFileName   = "bundle.json"
	ManifestFileName = "manifest.json"
)

// DefaultRunDir is the local-tier parent an unnamed run is parked under.
const DefaultRunDir = ".abcd/.work.local/scratch/reading-runs"

// MaxFileBytes bounds one admitted file. A file past the cap is a refusal, not
// a truncation: a silently shortened item would be an assembled input no re-run
// could reproduce from the manifest's hash.
const MaxFileBytes = 4 << 20

// LintConfigPath is the record-lint configuration the record scan reads its
// stores from. Enumeration comes from that scan and nowhere else: there is one
// parser of the record's shape in this binary.
const LintConfigPath = ".abcd/record-lint.json"

// lintRecordSchemaRule is the rule whose configuration names the record stores.
const lintRecordSchemaRule = "record_schema"

var (
	// targetRe is the whole grammar of the second operand besides "HEAD".
	targetRe = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	// itemKeyRe is the bundle key's shape: an ordinal, never a location.
	itemKeyRe = regexp.MustCompile(`^itm-[0-9]{4}$`)
)

// storeNodeType maps an include row's record-store prefix to the node type the
// record graph reports. It is a translation, not a second declaration of which
// stores exist: the graph owns that, and an entry here for a store no row names
// would be a claim this package reads a family it structurally cannot.
var storeNodeType = map[string]string{
	"itd": "intent",
	"spc": "spec",
	// The reading store, reached only by the candidate row at the comparative
	// position. Its BUCKET is the run directory, which is what lets the assembly
	// narrow that row to the derived run by setting Row.Bucket rather than by
	// growing a second selector (adr-2609021016272867).
	issueschema.ReadingItemFamily: "reading",
}

// rowClass is how a committed entry's object set narrows the row that admitted
// a candidate. It is derived from the row's own declaration and never from the
// candidate's path: the table says which rows enumerate a record store and
// which reach the repository root, and reading that off a path again is how two
// derivations of one fact come to disagree.
type rowClass int

const (
	// rowConstraint is a row that is neither a record store nor the tree: the
	// brief's chapters and the glossary. No part of the object set narrows a
	// constraint source, so such a row is admitted by kind alone.
	rowConstraint rowClass = iota
	// rowRecord is a row routed through the record graph — the shipped, drafts,
	// planned, specs and disciplines rows.
	rowRecord
	// rowTree is a row sourced at the repository root: the doc, config, source
	// and test rows.
	rowTree
)

// classOf reads one row's narrowing class off its own declaration.
func classOf(row Row) rowClass {
	if row.Store != "" {
		return rowRecord
	}
	if path.Clean(row.Source) == "." {
		return rowTree
	}
	return rowConstraint
}

// candidate is one admitted (file, field) pair before it is keyed.
type candidate struct {
	path     string
	field    string
	fieldIdx int
	kind     Kind
	// rowIdx and rowClass are the admitting row's identity and its narrowing
	// class, carried so the committed entry's object set can narrow per ROW —
	// a record row is narrowed to the object set's records only when the object
	// set reaches that row at all, and that question is a row's, not an item's
	// (spc-2609020626048722).
	rowIdx   int
	rowClass rowClass
	// scan is the admitting row's Scan, carried through so the manifest can
	// state per item whether the exclusion floor examined it. It is set from
	// the row and never inferred from the path: the row is the declaration, and
	// a second derivation of the same fact is how admission and examination
	// came to disagree (itd-194).
	scan Scan
	text string
}

// Assemble walks the repository under the include table at the given position
// and produces the assembled input and its manifest.
func Assemble(req AssembleRequest) (AssembleResult, error) {
	position, err := ParsePosition(string(req.Position))
	if err != nil {
		return AssembleResult{}, err
	}
	if req.RepoRoot == "" {
		return AssembleResult{}, fmt.Errorf("reading: no repository root given")
	}
	target, err := resolveTarget(req.RepoRoot, req.Target)
	if err != nil {
		return AssembleResult{}, err
	}
	if err := refuseSelfAdmittingOutDir(req.RepoRoot, req.OutDir, outDirLabel(req)); err != nil {
		return AssembleResult{}, err
	}

	// The comparative position derives its candidate run from the RECORD, before
	// anything is resolved or collected.
	//
	// The position refused outright until adr-2609021016272867: its object is
	// the widening reading's pre-admission output, and rather than hand it the
	// detection corpus and let it read the wrong object with every gate green,
	// itd-199 refused and scoped the channel out. The channel is this. No
	// operand names the run — the invocation is a position and a target state —
	// so the rule is the ADR's: the one committed widening run at the target
	// whose items carry no disposition and no admission, with none or more than
	// one refusing and listing the runs.
	var candidateRun WideningRun
	if position == PositionComparative {
		candidateRun, err = DeriveCandidateRun(req.RepoRoot, target)
		if err != nil {
			return AssembleResult{}, err
		}
	}

	presets, err := LoadPresets(req.RepoRoot)
	if err != nil {
		return AssembleResult{}, fmt.Errorf("reading: %w", err)
	}
	// The entry follows from the POSITION and from what is committed. No
	// operand names it, so nothing an operator typed can change what this run
	// is handed (adr-2609021016286571).
	entry, err := PresetFor(presets, position)
	if err != nil {
		return AssembleResult{}, fmt.Errorf("reading: %w", err)
	}
	applied := entry.Applied()

	cands, err := collect(req.RepoRoot, position, candidateRun.ID)
	if err != nil {
		return AssembleResult{}, err
	}
	if err := refuseDirtyIncludedPaths(req.RepoRoot, position, cands); err != nil {
		return AssembleResult{}, err
	}

	exclusions := ExclusionsFor(position)
	if err := assertExclusionsHook(cands, exclusions); err != nil {
		return AssembleResult{}, err
	}
	// The fail-closed half of the readings-store signal row, run over the
	// unfiltered walk beside the directory assertion for the same reason: a
	// narrow entry must not be able to quiet a breach a wide one would catch.
	if position == PositionComparative {
		if err := assertCandidateProjection(cands, candidateRun.ID); err != nil {
			return AssembleResult{}, err
		}
	}

	// The preset filter runs LAST, after the dirty gate and the exclusion
	// assertion have both run over the unfiltered walk. That ordering is
	// load-bearing rather than incidental: every structural property the
	// assembler holds — the deny, the floor, the tracked-set intersection, the
	// dirty gate — is a property of what the POSITION admits, and an entry must
	// not be able to shrink the set those gates examine. A narrow entry
	// therefore cannot quiet a dirty-tree refusal or an exclusion breach that
	// a wide one would have caught (spc-69).
	//
	// Within the entry the kinds ADMIT and the object set NARROWS, and the
	// record half of that narrowing is a question about a ROW rather than about
	// an item: a record row is narrowed to the object set's records when the
	// object set names any record under that row's source, and admitted whole
	// when it names none. So the rows the object set reaches are decided first,
	// over the whole unfiltered walk, and the per-item filter reads that answer
	// (spc-2609020626048722).
	byRow := map[int][]string{}
	for _, c := range cands {
		if c.rowClass == rowRecord {
			byRow[c.rowIdx] = append(byRow[c.rowIdx], c.path)
		}
	}
	narrows := make(map[int]bool, len(byRow))
	for rowIdx, paths := range byRow {
		// The drafts and planned rows are the exception, ruled on 2026-09-02:
		// they narrow to the records the entry NAMES whatever else the object
		// set names, so a draft or a planned intent travels only when the
		// committed entry names it. The companion's section 6.2 makes them
		// admissible here, and admissible is a permission rather than a scope —
		// the switch is where an entry takes that permission whole.
		if narrowsToNamedRecordsAlways(Table[rowIdx]) {
			narrows[rowIdx] = !entry.admitsEveryDraftAndPlanned()
			continue
		}
		narrows[rowIdx] = entry.namesRecordIn(paths)
	}
	scoped := make([]candidate, 0, len(cands))
	for _, c := range cands {
		// A CANDIDATE is not selected by an entry and never has been. An entry
		// names repository material — an object set of records and paths, and the
		// kinds within it — and the candidate set is selected by the derived run;
		// `candidate` is refused as a preset kind for exactly that reason, so
		// running these items through the kind filter would drop every one of
		// them (spc-2609020626039834, "The candidate row").
		if c.kind == KindCandidate || entry.selects(c, narrows[c.rowIdx]) {
			scoped = append(scoped, c)
		}
	}
	if len(scoped) == 0 {
		return AssembleResult{}, fmt.Errorf("reading: the committed entry for the %s position "+
			"selects no item that position admits, so there is nothing to assemble; an empty "+
			"assembly is a refusal rather than a bundle a reader would take for its whole object. "+
			"Widen the entry in %s", position, PresetConfigPath)
	}
	cands = scoped

	// The comparative position's two remaining facts: the criteria it
	// characterises against, and whether it is exercised at all.
	var criteria []string
	notExercised := false
	if position == PositionComparative {
		criteria, err = comparativeCriteria(cands)
		if err != nil {
			return AssembleResult{}, err
		}
		if got := candidateItems(cands); len(got) != candidateRun.Items {
			return AssembleResult{}, fmt.Errorf("reading: the derived run %s holds %d item(s) in the "+
				"ledger and the candidate row enumerated %d (%s); the manifest states the run's item "+
				"count as the count of candidates, and a disagreement means the two readers of one "+
				"run see different records", candidateRun.ID, candidateRun.Items, len(got),
				strings.Join(got, ", "))
		}
		// The fixed interpretation, ruled in advance and recorded before any
		// reading ran: where the widening reading returns fewer than two
		// configurations, the comparative reading has nothing to compare and is
		// not exercised (the 2026-09-02 interpretations entry; companion 7.6).
		// The bundle carries no candidate item, and the run is still staged, so
		// the outcome of a widening run is ONE shape — a committed comparative
		// run naming it — whether the position ran or not.
		if candidateRun.Items < 2 {
			notExercised = true
			cands = withoutCandidates(cands)
		}
	}

	runID, err := mintRunID()
	if err != nil {
		return AssembleResult{}, fmt.Errorf("reading: minting a run id: %w", err)
	}

	bundle := Bundle{
		Type:          BundleType,
		SchemaVersion: SchemaVersion,
		Position:      position,
		Preset:        bundlePreset(applied),
		Items:         make([]BundleItem, 0, len(cands)),
	}
	presetHash, err := applied.Hash()
	if err != nil {
		return AssembleResult{}, err
	}
	manifest := Manifest{
		Type:             ManifestType,
		SchemaVersion:    SchemaVersion,
		RunID:            runID,
		Position:         position,
		TargetCommit:     target,
		AssemblerVersion: AssemblerVersion(),
		Preset:           applied,
		PresetHash:       presetHash,
		Items:            make([]ManifestItem, 0, len(cands)),
		Exclusions:       exclusions,
	}
	if position == PositionComparative {
		exercised := !notExercised
		manifest.CandidateRun = candidateRun.ID
		manifest.CandidateRunTarget = candidateRun.Target
		manifest.Candidates = candidateRun.Items
		manifest.Exercised = &exercised
		manifest.CandidateFields = append([]string{}, CandidateFields...)
		manifest.Criteria = criteria
	}
	for i, c := range cands {
		key := fmt.Sprintf("itm-%04d", i+1)
		item := BundleItem{ItemKey: key, Kind: c.kind, Text: c.text}
		mItem := ManifestItem{
			ItemKey: key, Path: c.path, Field: c.field, Kind: c.kind, Scan: c.scan,
			Bytes: len(c.text), SHA256: sha256Hex([]byte(c.text)),
		}
		// A candidate is told which rdi-N it belongs to and which of the two
		// fields it is, and nothing else. Neither is a repository path, and both
		// stay empty on every other kind of item.
		if c.kind == KindCandidate {
			id := candidateIDOf(c.path)
			item.Candidate, item.Field = id, c.field
			mItem.Candidate = id
		}
		bundle.Items = append(bundle.Items, item)
		manifest.Items = append(manifest.Items, mItem)
	}

	hash, err := ManifestHash(manifest)
	if err != nil {
		return AssembleResult{}, err
	}
	res := AssembleResult{
		RunID:            runID,
		Position:         position,
		TargetCommit:     target,
		AssemblerVersion: AssemblerVersion(),
		ItemCount:        len(bundle.Items),
		ManifestHash:     hash,
		Preset:           applied,
		Size:             sizeReport(cands, position, PresetWindow(presets, position)),
		CandidateRun:     candidateRun.ID,
		Candidates:       candidateRun.Items,
		NotExercised:     notExercised,
		Artefacts:        []string{},
		Bundle:           bundle,
		Manifest:         manifest,
	}

	outDir, writeIt := resolveOutDir(req, runID)
	res.OutDir = outDir
	if !writeIt {
		return res, notExercisedError(notExercised, candidateRun)
	}
	if err := writeArtefacts(req.RepoRoot, outDir, outDirLabel(req), bundle, manifest); err != nil {
		return AssembleResult{}, err
	}
	res.Written = true
	res.Artefacts = []string{BundleFileName, ManifestFileName}
	return res, notExercisedError(notExercised, candidateRun)
}

// PositionNotExercised is the fixed interpretation as a refusal: the derived
// widening run holds fewer than two candidates, so the comparative reading has
// nothing to compare.
//
// It is returned BESIDE a populated result, because both halves are true — the
// assembly refused, and it staged the run whose ingest records the outcome. The
// front doors render the result and then exit on the error, which is the shape
// `reading ingest` already carries for a refusal that produced a durable record.
type PositionNotExercised struct {
	CandidateRun string
	Candidates   int
}

func (e *PositionNotExercised) Error() string {
	return fmt.Sprintf("the widening run %s returned %d configuration(s), so the comparative "+
		"reading has nothing to compare and this position is NOT EXERCISED. That is the "+
		"interpretation fixed in advance, before any reading ran: where the widening reading "+
		"returns fewer than two configurations the comparative reading is not exercised, and the "+
		"outcome is recorded as a committed comparative run with an empty item set naming that "+
		"widening run. The run is staged with an empty candidate set; ingest it to commit that "+
		"outcome", e.CandidateRun, e.Candidates)
}

// notExercisedError returns the refusal when the position was not exercised, and
// nil otherwise, so the two return sites above read as one decision.
func notExercisedError(notExercised bool, run WideningRun) error {
	if !notExercised {
		return nil
	}
	return &PositionNotExercised{CandidateRun: run.ID, Candidates: run.Items}
}

// comparativeCriteria reads the declared slate off the discipline item the
// assembly is about to hand over.
//
// A comparative bundle without the criteria characterises against nothing, and
// the criteria are never supplied at invocation (itd-191's gate), so an assembly
// that selected no discipline item REFUSES rather than assembling a reading that
// would have to invent them.
func comparativeCriteria(cands []candidate) ([]string, error) {
	for _, c := range cands {
		if c.kind == KindDiscipline && pathNamesRecord(c.path, CriteriaDiscipline) {
			return declaredCriteria(c.text)
		}
	}
	return nil, fmt.Errorf("reading: the comparative assembly selected no item from %s, which "+
		"declares the selection criteria; the criteria come from the record and never from the "+
		"invocation, and a comparative reading with no criteria characterises against nothing "+
		"(itd-191). Name %s in the object set of the comparative entry in %s",
		CriteriaDiscipline, CriteriaDiscipline, PresetConfigPath)
}

// withoutCandidates drops the candidate items, leaving the repository material
// the entry selected. It is what a not-exercised run stages: a bundle with no
// candidate item, on the same rules as any other assembly.
func withoutCandidates(cands []candidate) []candidate {
	out := make([]candidate, 0, len(cands))
	for _, c := range cands {
		if c.kind == KindCandidate {
			continue
		}
		out = append(out, c)
	}
	return out
}

// outDirLabel is the spelling a refusal quotes back: the operator's own, where
// the caller supplied one, and otherwise the directory itself.
func outDirLabel(req AssembleRequest) string {
	if req.OutDirLabel != "" {
		return req.OutDirLabel
	}
	return req.OutDir
}

// resolveOutDir decides where the two artefacts go and whether they are written
// at all. A dry run with no operator-named directory writes nothing, on the
// render-without-writing idiom `disembark plan` carries.
func resolveOutDir(req AssembleRequest, runID string) (string, bool) {
	if req.OutDir != "" {
		return req.OutDir, true
	}
	if req.DryRun {
		return "", false
	}
	return DefaultRunDir + "/" + runID, true
}

// writeArtefacts writes the assembled input and the manifest as two separate
// files. A relative output directory is taken against the repository root; an
// absolute one is used as given.
func writeArtefacts(repoRoot, outDir, label string, b Bundle, m Manifest) error {
	dir := outDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoRoot, filepath.FromSlash(outDir))
	}
	// The label is the OPERATOR's spelling, and there is none for the default run
	// directory — the assembler chose it. Falling back to the directory itself
	// keeps the refusal from naming nothing at all.
	if label == "" {
		label = outDir
	}
	if err := requireEmptyDir(label, dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("reading: creating the run directory: %w", err)
	}
	bundleRaw, err := EncodeBundle(b)
	if err != nil {
		return err
	}
	manifestRaw, err := EncodeManifest(m)
	if err != nil {
		return err
	}
	return writePair(dir, bundleRaw, manifestRaw)
}

// writePair writes the two artefacts, and leaves either both or neither.
//
// Each write is atomic on its own, which is not the same as the pair being
// atomic. A manifest write that fails after the bundle landed leaves a run that
// is half evidence — and worse, a directory that refuses every later run for
// being non-empty, so the failure is permanent until someone clears it by hand.
// The bundle is removed on that path, which puts the directory back the way the
// run found it.
func writePair(dir string, bundleRaw, manifestRaw []byte) error {
	bundlePath := filepath.Join(dir, BundleFileName)
	if err := fsutil.WriteFileAtomic(bundlePath, bundleRaw, 0o644); err != nil {
		return fmt.Errorf("reading: writing the assembled input: %w", err)
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, ManifestFileName), manifestRaw, 0o644); err != nil {
		os.Remove(bundlePath)
		return fmt.Errorf("reading: writing the manifest: %w", err)
	}
	return nil
}

// requireEmptyDir refuses an output directory that already holds something. The
// two artefacts are one run's evidence, and dropping them beside another run's
// leaves a directory whose manifest describes only half of what is in it.
func requireEmptyDir(named, dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading: reading the output directory: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("reading: the output directory %s is not empty (%d entr(y|ies)); one run's "+
			"artefacts are one run's evidence, so name an empty or absent directory", named, len(entries))
	}
	return nil
}

// refuseSelfAdmittingOutDir refuses an output directory the include table can
// reach. Writing a run where the table admits it is committing the NEXT run's
// contamination: the artefacts land as ordinary files, a later commit puts them
// in the tree, and ruling (18) is breached by a path the operator chose one run
// earlier. The refusal belongs at the moment the directory is named.
//
// Only a directory inside the repository can be reached, so an output path that
// resolves outside it is always fine.
func refuseSelfAdmittingOutDir(repoRoot, outDir, label string) error {
	if outDir == "" {
		return nil
	}
	abs := outDir
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(repoRoot, filepath.FromSlash(outDir))
	}
	// Both sides are resolved through their symlinks before being compared. A
	// lexical comparison reads a link by its NAME, so a directory named outside
	// the table's reach whose target is inside it walks straight through — and a
	// repository root reached through a link never matches its own paths.
	realRoot, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		realRoot = repoRoot
	}
	abs = resolveDeepest(abs)
	rel, err := filepath.Rel(realRoot, abs)
	if err != nil {
		return nil // not expressible against the repository, so not inside it
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return nil
	}
	for _, name := range []string{BundleFileName, ManifestFileName} {
		candidate := path.Join(rel, name)
		for _, p := range Positions() {
			if Admits(p, candidate) {
				return fmt.Errorf("reading: the output directory %s is inside the include table's reach "+
					"(%s would be admitted at the %s position), so a committed run would become a later "+
					"run's input; write outside the repository, or under %s", label, candidate, p, DefaultRunDir)
			}
		}
	}
	return nil
}

// resolveDeepest resolves the symlinks of the deepest ANCESTOR of p that exists,
// then re-joins the missing tail. An output directory is usually absent — that
// is the normal case — so resolving p itself would answer nothing about the
// links on the way to it.
func resolveDeepest(p string) string {
	missing := []string{}
	cur := p
	for {
		if resolved, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(append([]string{resolved}, missing...)...)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return p
		}
		missing = append([]string{filepath.Base(cur)}, missing...)
		cur = parent
	}
}

// resolveTarget validates the second operand and resolves it against HEAD.
//
// Assembly reads the WORKING TREE, so a target that is not the commit in front
// of the assembler is a refusal rather than a checkout: the manifest would
// otherwise name a commit whose content it never read.
func resolveTarget(repoRoot, target string) (string, error) {
	if target != "HEAD" && !targetRe.MatchString(target) {
		return "", fmt.Errorf("reading: target %q is neither HEAD nor a hexadecimal commit sha of 7 to 40 digits; "+
			"branch names and tags move, and the manifest's re-runnability rests on a reference that cannot", target)
	}
	head, err := gitutil.Run(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("reading: resolving HEAD: %w", err)
	}
	head = strings.TrimSpace(head)
	if head == "" {
		return "", fmt.Errorf("reading: HEAD does not resolve to a commit")
	}
	if target != "HEAD" && !strings.HasPrefix(head, target) {
		return "", fmt.Errorf("reading: HEAD is %s, which is not the target %s; assembly reads the working tree",
			head, target)
	}
	return head, nil
}

// refuseDirtyIncludedPaths refuses an assembly whose own input is uncommitted.
// Dirtiness elsewhere in the tree is not the assembler's business: a reading
// cannot see it, so it cannot contaminate the run.
//
// A path counts as included on either of two grounds, and it needs both. The
// assembly's own item paths catch an edit or an untracked addition. The include
// table's admissibility catches the case the item paths cannot: a file DELETED
// in the working tree is in neither the assembly nor the walk, yet it is part of
// the commit the manifest names, so a run over it would describe a target it
// never read.
//
// A third ground is neither of those: material an assembly READS FROM THE
// FILESYSTEM without ever putting it in a bundle. The two configuration files
// below decide what the walk collects, and the comparative position's fate
// families decide which run it collects; an uncommitted change to any of them
// reshapes the assembly exactly as an uncommitted change to an item does.
func refuseDirtyIncludedPaths(repoRoot string, position Position, cands []candidate) error {
	// -uall, not the default -unormal: git collapses an untracked DIRECTORY to a
	// single entry, and an admitted file inside a newly created directory would
	// then never be named — the prefix check below can only test paths the
	// assembly already holds, and an untracked file is not one of them.
	out, err := gitutil.RunCapped(repoRoot, 8<<20, "status", "--porcelain=v1", "-z", "-uall")
	if err != nil {
		return fmt.Errorf("reading: reading the working-tree status: %w", err)
	}
	included := make(map[string]bool, len(cands))
	for _, c := range cands {
		included[c.path] = true
	}
	// The record configuration decides which stores the record scan reads, so an
	// uncommitted edit to it reshapes the assembly as surely as an edit to a
	// record does. It sits under the deny, so no include row ever puts it in this
	// set; it is named here instead.
	included[LintConfigPath] = true
	// The preset configuration decides what a scope resolves to, so an
	// uncommitted edit to it reshapes the assembly exactly as an uncommitted
	// edit to the record configuration does. It sits under the deny too, so no
	// include row ever puts it in this set; it is named here for the same
	// reason and by the same argument (itd-199).
	included[PresetConfigPath] = true
	var dirty []string
	for _, entry := range dirtyPaths(out) {
		// The comparative derivation reads the two FATE families off the
		// filesystem — capture.ItemFate walks the dispositions and the admissions
		// directories to decide whether a widening run is still pre-admission
		// (adr-2609021016272867; companion 8.3, which sequences dispositioning
		// after the comparative reading). So an uncommitted fate selects a
		// different candidate run than the commit the manifest names holds: a
		// disposition deleted in the working tree makes a fated run look
		// pre-admission, and an admission added there disqualifies a run the
		// commit still admits. Neither family is admitted by an include row at any
		// position — a fate is the researcher's judgement and never a reading's
		// input — so no include row can put it in the set above, and it is named
		// here for the reason the two configuration files are.
		if position == PositionComparative && underFateFamily(entry) {
			dirty = append(dirty, entry)
			continue
		}
		if strings.HasSuffix(entry, "/") {
			for p := range included {
				if strings.HasPrefix(p, entry) {
					dirty = append(dirty, p)
				}
			}
			continue
		}
		if included[entry] || Admits(position, entry) {
			dirty = append(dirty, entry)
		}
	}
	if len(dirty) == 0 {
		return nil
	}
	sort.Strings(dirty)
	return fmt.Errorf("reading: %d included path(s) are uncommitted, starting with %s; "+
		"a dirty tree cannot be described by a commit reference, so the manifest would promise "+
		"a re-run it could not deliver", len(dirty), dirty[0])
}

// fateFamilyRoots names the two ledger families a fate is recorded in, derived
// from the ledger's own constants rather than written out: a family whose
// directory name moves must move here with it, or this gate would go on
// watching a path nothing writes to.
func fateFamilyRoots() [2]string {
	return [2]string{
		capture.LedgerRelPath + "/" + issueschema.DispositionsDir + "/",
		capture.LedgerRelPath + "/" + issueschema.AdmissionsDir + "/",
	}
}

// underFateFamily reports whether one dirty entry touches either fate family.
//
// A DIRECTORY entry counts on either side of the boundary: inside a family, and
// containing one. `-uall` collapses no untracked directory, but a status entry
// can still name a directory, and a family that arrives or disappears whole
// changes what the derivation reads just as surely as a single record does.
func underFateFamily(entry string) bool {
	for _, root := range fateFamilyRoots() {
		if strings.HasPrefix(entry, root) {
			return true
		}
		if strings.HasSuffix(entry, "/") && strings.HasPrefix(root, entry) {
			return true
		}
	}
	return false
}

// dirtyPaths parses `git status --porcelain=v1 -z` into the paths it reports.
//
// The -z form is what makes this parseable: it emits each entry NUL-terminated
// and never quotes or escapes a path, so a filename holding a space, a quote or
// a newline arrives verbatim and core.quotepath cannot change the format under
// the parser. A rename or copy entry carries its source as the following
// record, which is consumed with it.
func dirtyPaths(out string) []string {
	records := strings.Split(out, "\x00")
	var paths []string
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) < 4 {
			continue
		}
		status := rec[:2]
		paths = append(paths, rec[3:])
		// A rename or copy carries its SOURCE as the following record, and either
		// status column can declare one: `R ` is a staged rename, ` R` a worktree
		// one. The source is the path that was in the target commit, so dropping
		// it loses exactly the file whose disappearance from the include set this
		// gate exists to catch.
		if status[0] == 'R' || status[0] == 'C' || status[1] == 'R' || status[1] == 'C' {
			i++
			if i < len(records) && records[i] != "" {
				paths = append(paths, records[i])
			}
		}
	}
	return paths
}

// assertExclusions is the fail-closed half of the exclusion floor: the manifest
// DECLARES what was refused, and this refuses to emit an item that contradicts
// the declaration. A floor a run can quietly violate is a disclosure, not a
// gate.
// assertExclusionsHook is the seam that makes the ORDER of this pipeline
// observable, and it exists because the order is a claim nothing else can
// falsify.
//
// The gates above run over the unfiltered walk deliberately: a scope must not
// be able to shrink the set they examine, or a narrow scope could quiet a
// breach a wide one would have caught. Every other way of testing that turned
// out to be untestable — the dirty gate's predicate is a pure function of the
// position, so it refuses under either order, and the exclusion floor's own
// paths are structurally denied, so no candidate can breach one to begin with.
// The claim was therefore true, load-bearing, and unfalsifiable, which is the
// shape itd-195 says to make executable or stop making.
//
// A test swaps this to record how many candidates the assertion was handed.
var assertExclusionsHook = assertExclusions

func assertExclusions(cands []candidate, exclusions []Exclusion) error {
	for _, e := range exclusions {
		if e.Signal != "directory" && e.Signal != "file" {
			continue
		}
		for _, c := range cands {
			if c.path == e.Detail || strings.HasPrefix(c.path, e.Detail+"/") {
				return fmt.Errorf("reading: item %s lies under the excluded %s %s, which the manifest asserts was refused",
					c.path, e.Signal, e.Detail)
			}
		}
	}
	return nil
}

// collect gathers every admitted (file, field) pair at the position, in
// lexicographic path order with each file's fields in the order its row names
// them. The first row that reaches a path owns the projection applied to it.
func collect(repoRoot string, position Position, candidateRun string) ([]candidate, error) {
	graph, err := loadGraph(repoRoot)
	if err != nil {
		return nil, err
	}
	tracked, err := trackedSet(repoRoot)
	if err != nil {
		return nil, err
	}
	claimed := map[string]bool{}
	var out []candidate

	exclusions := ExclusionsFor(position)
	for rowIdx, row := range Table {
		if !row.AdmittedAt(position) {
			continue
		}
		class := classOf(row)
		row = narrowRow(row, position, candidateRun)
		paths, err := rowPaths(repoRoot, row, graph)
		if err != nil {
			return nil, err
		}
		paths = narrowPaths(paths, row, position)
		for _, rel := range paths {
			if claimed[rel] || !tracked[rel] {
				continue
			}
			claimed[rel] = true
			raw, err := fsutil.ReadGuarded(filepath.Join(repoRoot, filepath.FromSlash(rel)), MaxFileBytes)
			if err != nil {
				return nil, fmt.Errorf("reading: %s: %w", rel, err)
			}
			// refuseOwnArtefact runs over EVERY admitted file whatever its row,
			// because the artefact tag is a byte signature and needs no parse.
			if err := refuseOwnArtefact(rel, raw); err != nil {
				return nil, err
			}
			// A candidate is a reading record at the WIDENING position, and the
			// row's own path cannot establish that: every position's records
			// live in one family. So each file the candidate row enumerates is
			// validated as a reading record and refused by name if it was
			// returned anywhere else (adr-2609021016272867).
			if row.Kind == KindCandidate {
				if err := refuseNonWideningCandidate(rel, raw); err != nil {
					return nil, err
				}
			}
			// The floor runs over the row's DECLARATION, not over the file's
			// extension. That is the whole of adr-56's third rule made
			// mechanical: the table says which rows the floor parses, and the
			// floor runs over exactly those, so admission and examination
			// cannot describe two different sets. A row the floor does not
			// parse passes its document through untouched and every item it
			// yields is marked unscanned in the manifest.
			doc := string(raw)
			if row.Scan == ScanParsed {
				doc, err = redactExcluded(rel, doc, exclusions)
				if err != nil {
					return nil, err
				}
			}
			if len(row.Fields) == 0 {
				out = append(out, candidate{
					path: rel, kind: row.Kind, rowIdx: rowIdx, rowClass: class,
					scan: row.Scan, text: doc,
				})
				continue
			}
			for i, field := range row.Fields {
				text, ok, err := projectField(rel, doc, field)
				if err != nil {
					return nil, err
				}
				if !ok {
					continue
				}
				out = append(out, candidate{
					path: rel, field: field, fieldIdx: i,
					kind: row.Kind, rowIdx: rowIdx, rowClass: class,
					scan: row.Scan, text: text,
				})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].path != out[j].path {
			return out[i].path < out[j].path
		}
		return out[i].fieldIdx < out[j].fieldIdx
	})
	return out, nil
}

// narrowRow applies the comparative position's two narrowings to one row before
// it is enumerated. Both run BEFORE the committed entry is applied, which is
// what makes them un-widenable: an entry intersects what the row admits, so a
// narrowing here is one no entry can undo.
//
//   - The CANDIDATE row is narrowed to the derived run's directory by setting
//     the row's Bucket. The rdi store's bucket IS the run directory, so this is
//     the row's existing selector rather than a second mechanism, and the row
//     then reaches one run and never the family.
//   - The DISCIPLINES row is narrowed to CriteriaDiscipline, so no committed
//     entry can widen the comparative material past the criteria. The
//     consequence is deliberate: at this position an entry naming a record or a
//     path selects nothing outside that one discipline.
//
// At every other position the row is returned unchanged.
func narrowRow(row Row, position Position, candidateRun string) Row {
	if position != PositionComparative {
		return row
	}
	if row.Kind == KindCandidate {
		row.Bucket = candidateRun
	}
	return row
}

// narrowPaths applies the disciplines narrowing, which cannot be expressed on
// the row itself: the row selects by extension and by store, and the narrowing
// is to ONE record. It is the same shape a record in an entry's object set
// takes, applied here rather than there so that no entry can undo it.
func narrowPaths(paths []string, row Row, position Position) []string {
	if position != PositionComparative || row.Kind != KindDiscipline {
		return paths
	}
	out := make([]string, 0, 1)
	for _, rel := range paths {
		if pathNamesRecord(rel, CriteriaDiscipline) {
			out = append(out, rel)
		}
	}
	return out
}

// artefactTags are the two strings that identify this assembler's own output.
// They are the SIGNATURE the refusal below scans for, so they are stated once
// and read as bytes rather than as a shape a parser has to agree about.
var artefactTags = []string{BundleType, ManifestType}

// refuseOwnArtefact refuses an admitted file that IS one of this assembler's own
// artefacts. Ruling (18) says the instrument's own output never becomes its
// input, and the path deny alone cannot keep that promise: a run written to a
// directory the table reaches, then committed, arrives as an ordinary item, its
// manifest's repository paths riding into the bundle text while the current
// run's manifest still asserts the exclusion.
//
// The check is CONTENT-SIGNED, not extension- and parse-shaped. An earlier form
// keyed on a `.json` name and a successful top-level unmarshal, which made it a
// spelling check: the same artefact committed as `.yaml` or `.toml`, behind a
// byte-order mark, wrapped in another object, wrapped in an array, or carrying a
// duplicate key was admitted whole. What identifies the artefact is the tag it
// carries, wherever it sits and whatever the file is called, so the tag is
// looked for in the bytes before anything is parsed.
//
// The parse stays on behind the scan, and its reach is narrower than the scan's:
// it runs only on a `.json` name and reads only a TOP-LEVEL `_type`. What it
// adds is a tag spelled with a JSON unicode escape at that one position. An
// escaped tag inside a `.yaml` or `.toml` file, or nested in an object or an
// array, is caught by neither — a hand-rewritten artefact, which is a different
// threat from a run committed by accident, and the shape ruling (18) is actually
// about.
func refuseOwnArtefact(rel string, raw []byte) error {
	for _, tag := range artefactTags {
		if bytes.Contains(raw, []byte(tag)) {
			return ownArtefactError(rel, tag)
		}
	}
	if !strings.EqualFold(path.Ext(rel), ".json") {
		return nil
	}
	var probe struct {
		Type string `json:"_type"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil // not a JSON document, so its bytes are the whole of the evidence
	}
	for _, tag := range artefactTags {
		if probe.Type == tag {
			return ownArtefactError(rel, tag)
		}
	}
	return nil
}

// ownArtefactError states the refusal and the remedy.
func ownArtefactError(rel, tag string) error {
	return fmt.Errorf("reading: %s carries this assembler's own artefact tag %q, and the instrument's "+
		"own output never becomes its input; move it outside the include table's reach", rel, tag)
}

// requireConfiguredStores refuses a configuration that is silent about a store
// the include table names. It is the same refusal as an absent configuration,
// arriving one level in: an unnamed store contributes nothing to the record
// scan, and a row enumerating nothing is a hole the run would not report.
func requireConfiguredStores(repoRoot string, cfg lint.Config) error {
	configured := cfg.Rules[lintRecordSchemaRule].RecordStores
	for _, row := range Table {
		if row.Store == "" {
			continue
		}
		dir, ok := configured[row.Store]
		if !ok {
			return fmt.Errorf("reading: %s names no %q record store, so the include row %q would "+
				"enumerate nothing", LintConfigPath, row.Store, row.Source)
		}
		// A key present is not a store present. A retarget — a typo, a rename the
		// configuration did not follow — leaves the key in place and points it at
		// nothing, and the scan then reports an empty store exactly as it reports
		// a store with no records.
		// Lstat, not Stat: Stat follows a link, so a store reached through one
		// passes this check and then enumerates nothing, because the record scan
		// walks the real path. A link is refused rather than followed.
		info, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(dir)))
		// The READING store is provisioned on first use, so its absence is a
		// state and not a retarget: a repository that has commissioned no reading
		// has no readings directory, exactly as an empty lifecycle bucket is a
		// legitimate state of a record store. It is also not a silent hole, which
		// is what this check exists to close — the comparative position derives
		// its candidate run from the record and refuses BY NAME when no widening
		// run qualifies, listing what there is. A store pointed at something that
		// exists and is not a directory is still refused below, for every store
		// alike (adr-2609021016272867).
		if os.IsNotExist(err) && row.Store == issueschema.ReadingItemFamily {
			continue
		}
		if err != nil || !info.IsDir() {
			return fmt.Errorf("reading: %s points the %q record store at %s, which is not a directory "+
				"(a symlink is not followed), so the include row %q would enumerate nothing",
				LintConfigPath, row.Store, dir, row.Source)
		}
	}
	return nil
}

// trackedSet is the file set the target commit actually carries, read from git
// rather than from the filesystem.
//
// The walk and the dirty gate ask two different sources, and the gap between
// them is exactly the class of file the manifest's re-runnability rests on: a
// GITIGNORED file matching an include row is on disk, so a filesystem walk
// passes it, and `git status` says nothing about it, so the dirty gate cannot
// refuse it. Build output, a virtual environment and a vendored tree all land
// there, and an auditor re-running the assembly in a clean clone would get a
// different bundle under a different hash.
//
// Intersecting the walk with the tracked set closes that, and closes a
// submodule's inner content with it: git reports a gitlink, never the files
// beneath it, so they are absent from the tracked set by construction. An
// untracked file that is NOT ignored stays a refusal rather than a silent
// omission — the dirty gate sees it, and a genuine divergence from the target
// commit must be said out loud.
func trackedSet(repoRoot string) (map[string]bool, error) {
	files, err := gitutil.TrackedFiles(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("reading: listing the tracked files: %w", err)
	}
	set := make(map[string]bool, len(files))
	for _, f := range files {
		set[f] = true
	}
	return set, nil
}

// loadGraph reads the record corpus once, through the record_schema rule's own
// scan. A second parser of the record's shape would drift the moment a store
// gained a bucket, so there is not one.
func loadGraph(repoRoot string) (lint.RecordGraph, error) {
	lintRoot, err := os.OpenRoot(repoRoot)
	if err != nil {
		return lint.RecordGraph{}, fmt.Errorf("reading: opening the repository root: %w", err)
	}
	cfg, err := lint.LoadConfigInRoot(lintRoot, LintConfigPath)
	closeErr := lintRoot.Close()
	if err != nil {
		// A missing configuration is a REFUSAL, not an empty result. Without it
		// every record row contributes nothing and the run reports a clean
		// assembly of a reading that saw none of the record it exists to read
		// against — a silence indistinguishable from a repository with no record
		// at all.
		if os.IsNotExist(err) {
			return lint.RecordGraph{}, fmt.Errorf("reading: %s is absent, so the record scan enumerates "+
				"nothing and every record the include table names would be silently missing", LintConfigPath)
		}
		return lint.RecordGraph{}, fmt.Errorf("reading: loading %s: %w", LintConfigPath, err)
	}
	if closeErr != nil {
		return lint.RecordGraph{}, fmt.Errorf("reading: closing the repository root: %w", closeErr)
	}
	if err := requireConfiguredStores(repoRoot, cfg); err != nil {
		return lint.RecordGraph{}, err
	}
	graph, err := lint.LoadRecordGraph(cfg, repoRoot)
	if err != nil {
		return lint.RecordGraph{}, fmt.Errorf("reading: scanning the record: %w", err)
	}
	return graph, nil
}

// rowPaths resolves one row to the repo-relative files it admits, sorted.
func rowPaths(repoRoot string, row Row, graph lint.RecordGraph) ([]string, error) {
	var paths []string
	if row.Store != "" {
		nodeType, ok := storeNodeType[row.Store]
		if !ok {
			return nil, fmt.Errorf("reading: include row %q names an unknown record store %q", row.Source, row.Store)
		}
		for _, n := range graph.Nodes {
			if n.Type != nodeType {
				continue
			}
			if row.Bucket != "" && n.Lifecycle != row.Bucket {
				continue
			}
			if row.Reaches(n.Path) {
				paths = append(paths, n.Path)
			}
		}
		sort.Strings(paths)
		return paths, nil
	}

	base := repoRoot
	if row.Source != "." {
		base = filepath.Join(repoRoot, filepath.FromSlash(row.Source))
		// A walk row names a directory that must be there — a brief chapter, the
		// glossary. Absent, it enumerates nothing and the run reports clean, which
		// is the silent hole the store check already refuses one level up.
		//
		// A record store's BUCKET is deliberately not held to this: an empty
		// lifecycle bucket is a legitimate state of the record, and those rows
		// enumerate through the record graph rather than through this walk.
		info, err := os.Lstat(base)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("reading: the include row %q names %s, which is not a directory "+
				"(a symlink is not followed), so it would enumerate nothing", row.Source, row.Source)
		}
	}
	err := filepath.WalkDir(base, func(abs string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(repoRoot, abs)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			if !reachableDir(row, rel) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // a symlinked leaf is never followed
		}
		if row.Reaches(rel) {
			paths = append(paths, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading: walking %s: %w", row.Source, err)
	}
	sort.Strings(paths)
	return paths, nil
}

// reachableDir reports whether the walk may descend into a directory. It is the
// deny applied one level early, so a denied namespace is pruned rather than
// walked and discarded file by file.
func reachableDir(row Row, rel string) bool {
	sub := rel
	if row.Source != "." {
		src := path.Clean(row.Source)
		if rel == src {
			return true
		}
		if !strings.HasPrefix(rel, src+"/") {
			return false
		}
		sub = rel[len(src)+1:]
	}
	return !pathContainsDeniedSegment(sub) && !prefixDenied(rel)
}
