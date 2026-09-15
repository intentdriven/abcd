package reading

// scope.go is what a reading is ABOUT.
//
// The assembler shipped able to name a position and a commit and nothing else,
// which meant every reading received its position's whole corpus: about 9.2 MB
// and 2.4 million estimated tokens over this repository, with three of the four
// positions receiving a byte-identical item set. An instrument that cannot be
// pointed at anything cannot be given to a reader.
//
// A preset entry NARROWS and never widens. It intersects what the include table
// already admits at the position, so no entry reaches a row the table denies
// there, and the structural deny, the exclusion floor and the dirty gate all
// run over the unfiltered walk before this filter is applied (itd-199, spc-69).
//
// WHICH entry applies is not an operand. The design fixes the invocation at a
// position and a target state (framework v4 section 8.2 and ruling M8;
// companion v4 section 4.1), and adr-2609021016286571 supersedes adr-58
// accordingly: itd-199's scope operand is withdrawn, and the assembler applies
// the committed entry for the position it was invoked at. What the operand did
// is not lost — the presets already map each position to what it reads, and a
// preset is a record fact the repository supplies rather than something typed.
// Changing what a position reads is a commit to the preset file, reviewed and
// inside the dirty gate. There is no override at invocation and nothing to
// stamp, and no repository path is accepted at the invocation: a path may be
// named only inside a committed preset.

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// PresetConfigPath is the committed preset configuration. It is the ONE place a
// repository path may be named, and it joins the record-lint configuration in
// the dirty set by the same argument: an uncommitted edit to it reshapes an
// assembly as surely as an edit to a record does.
const PresetConfigPath = ".abcd/config/reading-presets.json"

// PresetSchemaVersion is the preset file's own shape version, separate from the
// artefacts' SchemaVersion: the configuration and the output are different
// shapes with different reasons to move.
//
// It goes 1 to 2 with the three-part entry: the object set the run is about, the
// kinds admitted within it, and the window the entry was calibrated for
// (spc-2609020626048722). The named `presets` map goes with the operand that
// chose between its keys; a version 2 file names one entry per position at the
// top level.
const PresetSchemaVersion = 2

// presetSchemaVersions is every version this loader reads. Version 1 goes on
// loading because the schema move is opt-in: an adopter's committed file keeps
// working until they move it, which is the whole of why this change's impact is
// `fix` and not `breaking` (cond-2609020626048715).
var presetSchemaVersions = map[int]bool{1: true, 2: true}

// TargetTokens is the size a committed entry AIMS at, stated and never
// enforced. The maintainer ruled on 2026-09-02 that any figure inside the
// reader's window is acceptable and that a figure over the target is stated to
// the operator, so nothing here refuses: the size report carries one line, and
// the entry's declaration and comment say what reader it was measured for
// (divergence register 24).
const TargetTokens = 200_000

// recordIDRe is the record-id token form. It is deliberately narrow: a token
// that is not one of these shapes is not a record id, and is never guessed at.
//
// It names ONLY the families the include table can admit. `adr-N` and `iss-N`
// were in it, and were dead: no row names a decisions store or an issue store,
// so those tokens validated, resolved, selected nothing, and ended in the
// selects-nothing refusal at every position — while the CLI help, the plugin
// page and the generated reference all advertised them. An affordance that can
// never work is worse than an absent one, because the operator's first move is
// to doubt their own invocation. If the table ever admits those families, this
// regex is where they come back.
var recordIDRe = regexp.MustCompile(`^(itd|spc)-[0-9]+$`)

// presetNameRe bounds a preset name to a shape that cannot be confused with a
// path, a record id or prose.
var presetNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]{0,31}$`)

// Selector is one clause of a scope. A candidate is selected if ANY clause
// matches it, so a scope is a union of its clauses and an empty scope selects
// nothing.
type Selector struct {
	// Kind selects every item of one material class.
	Kind Kind `json:"kind,omitempty"`
	// Record selects the items whose path names one record.
	Record string `json:"record,omitempty"`
	// Path selects the items at or beneath one repo-relative directory. It is
	// reachable ONLY from a committed preset, never from the invocation.
	Path string `json:"path,omitempty"`
}

// AppliedPreset is what one run was about: the committed entry for the invoked
// position, resolved to selectors.
//
// It carries no source token and no override stamp, and that absence is the
// point. Both were provenance about a CHOICE the operator made at the
// invocation, and there is no such choice: the entry is settled by the position
// and by what is committed (adr-2609021016286571). A run is reproducible from
// the commit the manifest names and the entry it records.
type AppliedPreset struct {
	// Selectors are the resolved clauses, in a canonical order so one entry
	// hashes to one value.
	Selectors []Selector `json:"selectors"`
}

// AppliedPreset carries no `selects` of its own, and its absence is the change
// spc-2609020626048722 makes to how an entry reads. A flat union of selectors
// said "kind OR record OR path", under which a path clause handed every kind
// beneath it and a record clause reached material the entry's kinds did not
// name. The entry's two axes are not a union: the KINDS admit and the OBJECT
// SET narrows, and that rule needs to know which include-table row admitted a
// candidate. So it lives on PositionEntry, and this type stays what it always
// was for the manifest — the applied entry rendered as clauses, in a canonical
// order, so one entry hashes to one value.

// pathNamesRecord reports whether a repo-relative path is the record's own
// file. A record's file is named for its id, so the basename either IS the id
// with an extension or begins with the id and a hyphen — the slug form every
// record store writes. A prefix test alone would make itd-19 select itd-198.
func pathNamesRecord(rel, id string) bool {
	base := path.Base(rel)
	if strings.HasPrefix(base, id+"-") {
		return true
	}
	return base == id+".md" || base == id+".json"
}

// underPath reports whether a repo-relative path lies at or beneath dir.
func underPath(rel, dir string) bool {
	dir = path.Clean(dir)
	if dir == "." {
		return true
	}
	return rel == dir || strings.HasPrefix(rel, dir+"/")
}

// canonicalise sorts an entry's selectors and drops duplicates, so two entries
// naming the same thing in a different order are one entry and hash alike.
func canonicalise(sels []Selector) []Selector {
	seen := make(map[Selector]bool, len(sels))
	out := make([]Selector, 0, len(sels))
	for _, s := range sels {
		if s == (Selector{}) || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Record != b.Record {
			return a.Record < b.Record
		}
		return a.Path < b.Path
	})
	return out
}

// Hash is the applied entry's content hash, so a manifest can name the entry a
// run actually had and a reader can tell two runs apart by it. It also means a
// preset edited later can never make a past run unreadable: the run names the
// entry it applied, not the file as it stands now.
func (s AppliedPreset) Hash() (string, error) {
	data, err := json.Marshal(struct {
		Selectors []Selector `json:"selectors"`
	}{Selectors: s.Selectors})
	if err != nil {
		return "", fmt.Errorf("hashing the applied reading preset: %w", err)
	}
	return sha256Hex(data), nil
}

// positionScopeV1 is one position's scope inside a version 1 preset. It is the
// shape spc-69 named, kept because a version 1 file goes on loading; it is read
// into a PositionEntry and never used past the load.
type positionScopeV1 struct {
	Kinds   []Kind   `json:"kinds"`
	Records []string `json:"records"`
	Paths   []string `json:"paths"`
}

// presetV1 holds a version 1 preset's per-position scopes.
//
// `extends` retired with the second preset name. It existed to make "warm is
// cold plus a delta" a property rather than a review note, and there is no
// second name for it to relate: one entry per position stands, and a repository
// that wants a wider reading commits a wider entry (adr-2609021016286571).
type presetV1 struct {
	Positions map[string]positionScopeV1 `json:"positions"`
}

// presetFileV1 is the committed configuration at schema version 1.
type presetFileV1 struct {
	SchemaVersion int                 `json:"schema_version"`
	Presets       map[string]presetV1 `json:"presets"`
}

// ObjectSet is what a run is ABOUT: which records and which delivered paths.
// The term is the design framework's own (section 13), and the two lists are
// one fact shared by every kind the entry admits.
type ObjectSet struct {
	Records []string `json:"records"`
	Paths   []string `json:"paths"`
}

// Window is the estimated-token figure an entry was calibrated to, beside the
// measurement it was taken from.
//
// TokensEst is the DECLARATION, on the size report's own byte-derived basis
// (bytes divided by 3.85, spc-68), and the eval holds every entry to it. The
// three measured_* values are DISCLOSURE and nothing gates on them beyond
// shape: MeasuredAt must match the assembler's target grammar, and its
// reachability is deliberately not checked, because a squash or rebase merge
// rewrites a branch sha out of existence and a disclosure that fails the build
// after one would teach people to omit it.
type Window struct {
	TokensEst         int    `json:"tokens_est"`
	MeasuredTokensEst int    `json:"measured_tokens_est"`
	MeasuredBytes     int    `json:"measured_bytes"`
	MeasuredAt        string `json:"measured_at"`
}

// PositionEntry is one position's committed entry, in the three parts the
// design names (spc-2609020626048722; divergence register 25).
//
// Comment is free text nothing reads except a reviewer; it is declared so the
// strict decoder admits it, and it is where the entry says why it names the
// kinds it does and what reader a figure over the target was measured for.
type PositionEntry struct {
	Comment string    `json:"comment,omitempty"`
	Object  ObjectSet `json:"object"`
	Kinds   []Kind    `json:"kinds"`
	Window  *Window   `json:"window,omitempty"`
	// AdmitDraftsAndPlanned hands the entailment position EVERY draft and
	// planned intent the include table admits there, rather than the object
	// set's alone. It is a pointer so that declaring it is distinguishable from
	// omitting it, which is what lets the loader refuse it at a position where
	// it means nothing rather than accept a key nobody reads.
	//
	// The maintainer ruled on 2026-09-02, at the Phase A review, that the
	// entailment reading is handed the object set only. The framework's section
	// 13 fixes the object set and the companion's section 6.2 makes drafts and
	// planned intents ADMISSIBLE there: admissible is a permission, the object
	// set is the scope, and handing every draft and planned intent in the
	// repository (147 projected intents on the tree the ruling was given over)
	// exceeds it. So the companion's admissibility becomes this switch — yes or
	// no for drafts and planned intents beyond the object set, default no — and
	// the drafts and planned rows otherwise narrow by the entry's record list
	// exactly as the shipped row does (divergence register 1 as corrected).
	AdmitDraftsAndPlanned *bool `json:"admit_drafts_and_planned,omitempty"`
}

// admitsEveryDraftAndPlanned reports whether the entry declares the switch on.
// An absent declaration is off, which is the default the ruling fixes.
func (e PositionEntry) admitsEveryDraftAndPlanned() bool {
	return e.AdmitDraftsAndPlanned != nil && *e.AdmitDraftsAndPlanned
}

// PresetFile is the committed configuration, one entry per position.
//
// A preset carries an entry PER POSITION rather than one every position shares,
// because the finding this exists to fix is that three of the four positions
// received a byte-identical item set, and one entry over four near-identical
// admissions reproduces it exactly (iss-2608311501240566).
type PresetFile struct {
	SchemaVersion int                      `json:"schema_version"`
	Positions     map[string]PositionEntry `json:"positions"`
}

// admitsKind reports whether the entry names one material kind. The kinds
// ADMIT: nothing outside the list travels, whatever the object set names.
func (e PositionEntry) admitsKind(k Kind) bool {
	for _, want := range e.Kinds {
		if want == k {
			return true
		}
	}
	return false
}

// selects reports whether one collected candidate is in this entry.
//
// The kinds admit and the object set narrows, and HOW it narrows depends on the
// include-table row that admitted the candidate:
//
//   - A record row (the shipped, drafts, planned, specs and disciplines rows)
//     is narrowed to the object set's records when the object set names any
//     record under that row's source — narrowRecords, decided per row by the
//     caller — and admitted whole when it names none. That is what hands a
//     definition's constraint sources whole while handing the spec and
//     intent-projection kinds for the object set's records alone. The drafts
//     and planned rows are the exception the caller applies: they narrow
//     ALWAYS, so a draft or a planned intent travels only when the entry names
//     it, unless the entry declares the admissibility switch (see
//     narrowsToNamedRecordsAlways and AdmitDraftsAndPlanned).
//   - A tree row (the doc, config, source and test rows at the repository root)
//     is narrowed to the files at or beneath the object set's paths, and an
//     entry with no path hands nothing from the tree whatever kinds it lists.
//   - Everything else — the brief chapters and the glossary — is admitted by
//     kind alone, because no part of the object set narrows a constraint
//     source.
func (e PositionEntry) selects(c candidate, narrowRecords bool) bool {
	if !e.admitsKind(c.kind) {
		return false
	}
	switch c.rowClass {
	case rowTree:
		for _, p := range e.Object.Paths {
			if underPath(c.path, p) {
				return true
			}
		}
		return false
	case rowRecord:
		if !narrowRecords {
			return true
		}
		for _, r := range e.Object.Records {
			if pathNamesRecord(c.path, r) {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// narrowsToNamedRecordsAlways reports whether a record row hands ONLY the
// records the entry names, whatever else the object set names.
//
// It is the drafts and planned rows and no other, and the ground is the ruling
// of 2026-09-02. Every other record row is admitted whole when the object set
// names none of its records, which is right for a constraint source: a spec or
// a discipline the object set does not name is context the reading reads
// AGAINST. A draft or a planned intent is not context — it is the claim record
// the entailment position reads, so an unnamed one is material the run is not
// about. Under the general rule the entailment entry handed every draft and
// planned intent in the repository, 147 projected intents on the tree the
// ruling was given over, where framework 13's object set is the fifteen
// workstream intents plus the ten Iteration 2 intents the record extends it by.
//
// The row is identified by its own declaration — the intent store's drafts and
// planned buckets — rather than by a path test, for the reason classOf gives:
// the table says what a row enumerates, and reading that off a path again is
// how two derivations of one fact come to disagree.
func narrowsToNamedRecordsAlways(r Row) bool {
	return r.Store == "itd" && (r.Bucket == "drafts" || r.Bucket == "planned")
}

// namesRecordIn reports whether the object set names a record whose file is one
// of the paths given. It is the per-row question selects takes as narrowRecords:
// the object set reaches this row, so this row narrows.
func (e PositionEntry) namesRecordIn(paths []string) bool {
	for _, p := range paths {
		for _, r := range e.Object.Records {
			if pathNamesRecord(p, r) {
				return true
			}
		}
	}
	return false
}

// Applied renders the entry as the canonical clause list the manifest and the
// bundle carry. It is provenance, not the filter: what the run applied, in one
// order, so a reader can tell two runs apart by the entry they had.
func (e PositionEntry) Applied() AppliedPreset {
	out := make([]Selector, 0, len(e.Kinds)+len(e.Object.Records)+len(e.Object.Paths))
	for _, k := range e.Kinds {
		out = append(out, Selector{Kind: k})
	}
	for _, r := range e.Object.Records {
		out = append(out, Selector{Record: r})
	}
	for _, raw := range e.Object.Paths {
		out = append(out, Selector{Path: path.Clean(raw)})
	}
	return AppliedPreset{Selectors: canonicalise(out)}
}

// LoadPresets reads and validates the committed preset configuration.
//
// It is strict on unknown fields, like every other artefact this package reads:
// a key nobody reads is a scope nobody applies, and a preset that silently does
// less than it says is the failure this whole intent exists to close.
func LoadPresets(repoRoot string) (PresetFile, error) {
	abs := filepath.Join(repoRoot, filepath.FromSlash(PresetConfigPath))

	// The preset's whole safety argument is that it is COMMITTED and reviewed:
	// adr-58 admits a preset name at the invocation on that ground alone, and
	// brief invariant 15 now says so. Neither half was checked, and the dirty
	// gate cannot supply it — git reports nothing for a file it ignores, so a
	// repository that gitignores `.abcd/` (which brief invariant 5 does for
	// public visibility) ran happily against an untracked preset and stamped
	// the run `overridden: false`, asserting "ran as reviewed" on an
	// examination that established only "git reported no modification". That is
	// an attestation exceeding its evidence, which brief invariant 16 forbids.
	// trackedSet reads the git INDEX, not HEAD, so on its own this establishes
	// "known to git" rather than "committed". The committed property is
	// delivered by this check together with the dirty gate downstream, which
	// refuses an added-but-uncommitted preset as an `A ` entry. Both are needed:
	// the dirty gate alone missed an ignored file, and this check alone would
	// admit a staged one.
	tracked, err := trackedSet(repoRoot)
	if err != nil {
		return PresetFile{}, err
	}
	if !tracked[PresetConfigPath] {
		return PresetFile{}, fmt.Errorf("%s is not tracked: a preset is admitted at the "+
			"invocation because it is committed and reviewed, so an untracked or ignored one "+
			"has no such standing and is refused rather than read", PresetConfigPath)
	}

	// And it must be a regular file. A committed SYMLINK is tracked, and git
	// reports nothing when its target changes, so the effective configuration
	// would be permanently unreviewed and freely mutable with every gate green.
	fi, err := os.Lstat(abs)
	if err != nil {
		return PresetFile{}, fmt.Errorf("reading %s: %w", PresetConfigPath, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return PresetFile{}, fmt.Errorf("%s is a symlink: its target is outside what the commit "+
			"records, so what it resolves to is not what was reviewed", PresetConfigPath)
	}
	if !fi.Mode().IsRegular() {
		return PresetFile{}, fmt.Errorf("%s is not a regular file", PresetConfigPath)
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return PresetFile{}, fmt.Errorf("reading %s: %w", PresetConfigPath, err)
	}

	// The version is read before the shape, because the two shapes are different
	// documents and a strict decoder cannot be pointed at both at once. A
	// version 1 file declaring a `window`, or a version 2 file carrying a
	// `presets` map, is therefore refused as an unknown field by the decoder for
	// the version it claims — which is what keeps the two shapes from being
	// mixed into a third nobody specified.
	var probe struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return PresetFile{}, fmt.Errorf("decoding %s: %w", PresetConfigPath, err)
	}
	if !presetSchemaVersions[probe.SchemaVersion] {
		return PresetFile{}, fmt.Errorf("decoding %s: schema_version is %d, want %s",
			PresetConfigPath, probe.SchemaVersion, joinVersions())
	}

	var pf PresetFile
	if probe.SchemaVersion == 1 {
		pf, err = decodeV1(raw)
	} else {
		pf, err = decodeV2(raw)
	}
	if err != nil {
		return PresetFile{}, err
	}
	if len(pf.Positions) == 0 {
		return PresetFile{}, fmt.Errorf("%s names no entry for any position", PresetConfigPath)
	}
	if err := validateEntries(pf); err != nil {
		return PresetFile{}, err
	}
	return pf, nil
}

// joinVersions renders the versions this loader reads, for a refusal that says
// which shapes exist rather than only which one it wanted.
func joinVersions() string {
	out := make([]string, 0, len(presetSchemaVersions))
	for v := range presetSchemaVersions {
		out = append(out, fmt.Sprintf("%d", v))
	}
	sort.Strings(out)
	return strings.Join(out, " or ")
}

// decodeV1 reads spc-69's named shape into the version 2 entry set.
//
// A version 1 file holding exactly one preset keeps working: its kinds, records
// and paths become the position entries, with the records and paths read as the
// object set and no window declared. A file holding more than one refuses,
// naming them, because nothing at the invocation can choose between them and
// the design admits no operand that could (cond-2609021004074586).
func decodeV1(raw []byte) (PresetFile, error) {
	if err := refuseDuplicateKeys(raw, "presets"); err != nil {
		return PresetFile{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var old presetFileV1
	if err := dec.Decode(&old); err != nil {
		return PresetFile{}, fmt.Errorf("decoding %s: %w", PresetConfigPath, err)
	}
	if len(old.Presets) == 0 {
		return PresetFile{}, fmt.Errorf("%s names no preset", PresetConfigPath)
	}
	if len(old.Presets) > 1 {
		return PresetFile{}, fmt.Errorf("%s names %d presets (%s); the invocation names none, so "+
			"there is nothing to choose between them. Commit one entry per position and keep the "+
			"other in the file's history", PresetConfigPath, len(old.Presets), joinPresets(old))
	}
	for name, p := range old.Presets {
		// The preset NAME is no longer a token anything resolves, so these three
		// refusals are about the FILE rather than about resolution: a key named
		// for a material kind or shaped like a record id reads, to a reviewer, as
		// the thing it is named after rather than as the entry set, and the one
		// file whose whole safety argument is that a human reviewed it is the
		// last place to leave a name that means two things.
		if err := validPresetName(name); err != nil {
			return PresetFile{}, err
		}
		out := PresetFile{SchemaVersion: 1, Positions: map[string]PositionEntry{}}
		for pos, ps := range p.Positions {
			out.Positions[pos] = PositionEntry{
				Kinds:  ps.Kinds,
				Object: ObjectSet{Records: ps.Records, Paths: ps.Paths},
			}
		}
		return out, nil
	}
	return PresetFile{}, fmt.Errorf("%s names no preset", PresetConfigPath)
}

// decodeV2 reads the current shape: one entry per position at the top level.
func decodeV2(raw []byte) (PresetFile, error) {
	if err := refuseDuplicateKeys(raw, "positions"); err != nil {
		return PresetFile{}, err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var pf PresetFile
	if err := dec.Decode(&pf); err != nil {
		return PresetFile{}, fmt.Errorf("decoding %s: %w", PresetConfigPath, err)
	}
	// Every entry declares the window it was calibrated for. A missing one
	// refuses by POSITION: an entry with no declaration is one the eval cannot
	// hold to anything, and a silent absence is how the instrument reached
	// release measured by nobody (spc-2609020626048722).
	for _, pos := range sortedPositionKeys(pf) {
		e := pf.Positions[pos]
		switch {
		case e.Window == nil:
			return PresetFile{}, fmt.Errorf("%s: the entry for %s declares no window; at "+
				"schema_version 2 every position states the window it was calibrated for",
				PresetConfigPath, pos)
		case e.Window.TokensEst <= 0:
			return PresetFile{}, fmt.Errorf("%s: the entry for %s declares a window of %d "+
				"estimated tokens; a declaration is a figure an assembly can be held to",
				PresetConfigPath, pos, e.Window.TokensEst)
		case e.Window.MeasuredTokensEst < 0 || e.Window.MeasuredBytes < 0:
			return PresetFile{}, fmt.Errorf("%s: the entry for %s declares a negative "+
				"measurement", PresetConfigPath, pos)
		case !targetRe.MatchString(e.Window.MeasuredAt):
			return PresetFile{}, fmt.Errorf("%s: the entry for %s names %q as the commit it was "+
				"measured on, which is not a commit sha of 7 to 40 hexadecimal digits",
				PresetConfigPath, pos, e.Window.MeasuredAt)
		}
	}
	return pf, nil
}

// sortedPositionKeys returns the file's position tokens in a stable order, so a
// refusal over a map names the same position on every run.
func sortedPositionKeys(pf PresetFile) []string {
	out := make([]string, 0, len(pf.Positions))
	for pos := range pf.Positions {
		out = append(out, pos)
	}
	sort.Strings(out)
	return out
}

// validPresetName holds a version 1 preset key to a shape that cannot be
// confused with a path, a record id, prose, or a material kind.
func validPresetName(name string) error {
	if !presetNameRe.MatchString(name) {
		return fmt.Errorf("%s: preset %q is not a valid name", PresetConfigPath, name)
	}
	for _, k := range Kinds() {
		if string(k) == name {
			return fmt.Errorf("%s: preset %q collides with the material kind of the same name; "+
				"one token may not mean two things", PresetConfigPath, name)
		}
	}
	if recordIDRe.MatchString(name) {
		return fmt.Errorf("%s: preset %q is shaped like a record id", PresetConfigPath, name)
	}
	return nil
}

// refuseDuplicateKeys refuses the named top-level object naming one key twice.
//
// Go's JSON decoder takes the LAST duplicate silently, and DisallowUnknownFields
// says nothing about duplicates. Against the one file whose entire safety
// argument is that a human reviewed it, silent last-wins is a review-evasion
// vector: a second `"detection"` block low in the file replaces the reviewed
// one, and a reviewer reading top-down sees the first.
//
// The container is named by the caller because the two schema versions put the
// keys in different places — `presets` at version 1, `positions` at version 2 —
// and one check over whichever container the version uses is better than two
// that can drift apart.
func refuseDuplicateKeys(raw []byte, container string) error {
	if !json.Valid(raw) {
		return nil // the strict decode below reports the real parse error
	}
	seen := map[string]int{}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	depth := 0
	inside := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case json.Delim:
			switch t {
			case '{':
				depth++
			case '}':
				depth--
				if depth <= 1 {
					inside = false
				}
			}
		case string:
			if depth == 1 && t == container {
				inside = true
				continue
			}
			if inside && depth == 2 {
				seen[t]++
				if seen[t] > 1 {
					return fmt.Errorf("%s names %q more than once under %q; the last would win "+
						"silently, so a reviewed block could be replaced by one further down the file",
						PresetConfigPath, t, container)
				}
			}
		}
	}
	return nil
}

// validateEntries refuses a configuration that could not mean one thing. It
// runs over the version 2 entry set, which a version 1 file has already been
// read into, so every refusal spc-69 named survives the schema move.
func validateEntries(pf PresetFile) error {
	kinds := make(map[Kind]bool, len(Kinds()))
	for _, k := range Kinds() {
		kinds[k] = true
	}
	for _, pos := range sortedPositionKeys(pf) {
		e := pf.Positions[pos]
		if _, err := ParsePosition(pos); err != nil {
			return fmt.Errorf("%s: %w", PresetConfigPath, err)
		}
		// The switch means something at ONE position. The drafts and planned
		// rows are admitted at entailment and nowhere else (the include table's
		// two rows; companion 6.2), so the key anywhere else is a permission
		// over material that position never sees — and a key that reads as
		// though it does something is exactly what the strict decoder refuses
		// everywhere else in this file.
		if e.AdmitDraftsAndPlanned != nil && Position(pos) != PositionEntailment {
			return fmt.Errorf("%s: the entry for %s declares admit_drafts_and_planned, which "+
				"means nothing there: the drafts and planned rows are admitted at the %s "+
				"position alone, so the switch is declared at %s or not at all",
				PresetConfigPath, pos, PositionEntailment, PositionEntailment)
		}
		// The comparative refusal is WITHDRAWN. itd-199's tenth criterion refused
		// an entry for that position because the position did not assemble;
		// adr-2609021016272867 gives it a channel, and its entry names the
		// repository material passed beside the candidates — which at that
		// position is the criteria discipline and nothing else.
		for _, k := range e.Kinds {
			if !kinds[k] {
				return fmt.Errorf("%s: the entry for %s names the unknown kind %q; "+
					"the vocabulary is closed", PresetConfigPath, pos, k)
			}
			// `candidate` is a member of the vocabulary and NOT a preset kind. An
			// entry selects repository material; a candidate is selected by the
			// derived widening run, and an entry naming the kind would read as
			// choosing the candidate set — which nothing at the invocation, and
			// nothing in a preset, may do (adr-2609021016272867).
			if k == KindCandidate {
				return fmt.Errorf("%s: the entry for %s names the kind %q, which no entry selects: "+
					"an entry names repository material, and the candidate set is DERIVED from the "+
					"record as the one committed widening run at the target whose items carry no "+
					"disposition and no admission", PresetConfigPath, pos, k)
			}
		}
		for _, r := range e.Object.Records {
			if !recordIDRe.MatchString(r) {
				return fmt.Errorf("%s: the entry for %s names %q, which is not a record id",
					PresetConfigPath, pos, r)
			}
		}
		for _, raw := range e.Object.Paths {
			if err := validPresetPath(raw); err != nil {
				return fmt.Errorf("%s: the entry for %s: %w", PresetConfigPath, pos, err)
			}
		}
	}
	return nil
}

// validPresetPath bounds the one place a repository path may be named. A preset
// cannot reach what the table denies — the filter runs after collection, so a
// denied path was never a candidate — but a path that LOOKS like it reaches
// there is refused at the door rather than silently selecting nothing.
func validPresetPath(raw string) error {
	if raw == "" {
		return fmt.Errorf("names an empty path")
	}
	if filepath.IsAbs(raw) || strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "~") {
		return fmt.Errorf("names the absolute path %q; preset paths are repo-relative", raw)
	}
	// A path carrying a control character renders into a terminal line, and
	// invariant 13 holds every runtime-read string to being sanitised before it
	// gets there. Refusing it at the door is better than sanitising it at every
	// render site that will ever exist.
	for _, r := range raw {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("names a path carrying a control character")
		}
	}
	if strings.TrimSpace(raw) != raw {
		return fmt.Errorf("names %q, which is surrounded by whitespace", raw)
	}
	if strings.Contains(raw, "\\") {
		return fmt.Errorf("names %q, which uses backslash separators; paths are POSIX and "+
			"repo-relative", raw)
	}
	clean := path.Clean(raw)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("names %q, which escapes the repository", raw)
	}
	// "." selects the whole repository, so it is a scope that narrows nothing.
	// A scope exists to narrow; an all-selecting path clause is either a
	// mistake or a way to spell "everything" that review would not read as one.
	if clean == "." {
		return fmt.Errorf("names %q, which selects the whole repository; a scope narrows, so "+
			"name the kinds you want rather than a path that excludes nothing", raw)
	}
	if pathContainsDeniedSegment(clean) || prefixDenied(clean) {
		return fmt.Errorf("names %q, which the structural deny covers; a preset cannot "+
			"reach what the table denies", raw)
	}
	return nil
}

// PresetFor returns the committed entry for one position.
//
// It takes no token, because there is none to take: the invocation is a
// position and a target state, and which entry applies follows from the
// position alone (adr-2609021016286571). A file naming no entry for the
// position refuses rather than defaulting to everything — a position served the
// whole corpus because its entry was forgotten is exactly the silent widening
// the presets exist to close.
func PresetFor(pf PresetFile, position Position) (PositionEntry, error) {
	e, ok := pf.Positions[string(position)]
	if !ok || len(e.Kinds) == 0 {
		return PositionEntry{}, fmt.Errorf("%s names no entry for the %s position, so a run there "+
			"would assemble nothing; an entry that selects nothing is a refusal rather than an "+
			"empty bundle. Commit an entry for %s", PresetConfigPath, position, position)
	}
	return e, nil
}

// PresetWindow returns the declaration the entry for one position carries, or
// nil where none is declared — which at schema version 2 cannot happen, and at
// version 1 always does. The size report says which of the two it is looking at
// rather than rendering a zero.
func PresetWindow(pf PresetFile, position Position) *Window {
	e, ok := pf.Positions[string(position)]
	if !ok {
		return nil
	}
	return e.Window
}

// joinPresets renders a version 1 file's preset names for a refusal, so a file
// holding more than one says which ones rather than only how many.
func joinPresets(old presetFileV1) string {
	out := make([]string, 0, len(old.Presets))
	for name := range old.Presets {
		out = append(out, name)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
