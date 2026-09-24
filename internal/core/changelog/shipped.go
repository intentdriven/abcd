package changelog

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The two record families a release is cut from. Directory-as-truth is the
// lifecycle authority in this repo — an intent's folder IS its status — so
// "what shipped" is exactly "which record files live in the terminal folders",
// and no separate release ledger is needed.
const (
	intentsShippedDir = ".abcd/development/intents/shipped"
	// Derived from the ledger root rather than spelled again: two literals for
	// one directory is how a gate ends up scanning a tree the writer no longer
	// uses (see issuesLedgerDir).
	issuesResolvedDir = issuesLedgerDir + "/resolved"
)

// recordPaths are the pathspecs handed to git ls-tree, in the order the sets are
// reported.
var recordPaths = []string{intentsShippedDir, issuesResolvedDir}

// recordFileRe matches a record filename and captures its id. The record
// families are the only files that count: a directory README or a stray note
// living beside them is not a release line. internal/core/lint holds equivalent
// unexported filename matchers for its own scanners (intentFileRe/issueFileRe);
// if a third consumer appears, that pair and this one want consolidating into a
// single exported matcher rather than a third copy.
var recordFileRe = regexp.MustCompile(`^((?:itd|iss)-\d+)[^/]*\.md$`)

// Record is one record in a release cut: where it lives, what it is called, and
// the one product judgement it declares.
type Record struct {
	// Path is the repo-relative, slash-separated path of the record file.
	Path string
	// ID is the record id derived from the FILENAME (itd-73, iss-24), not from
	// frontmatter: the filename is what the id-uniqueness lint already governs,
	// and issue frontmatter quotes its id, so the filename is the cheaper and
	// more consistent source.
	ID string
	// Impact is the parsed judgement, or "" when the record carries none.
	Impact Impact
	// ImpactErr is why the impact could not be parsed, empty when it parsed. A
	// record is reported as unlabelled rather than defaulted, because defaulting
	// a missing judgement silently under-bumps the release.
	ImpactErr string
	// Title names the record: its `# ` heading, else its frontmatter slug, else
	// its id. It is never empty (see summarise).
	Title string
	// Summary is the record's opening paragraph, collapsed and capped. Title and
	// Summary are the SOURCE MATERIAL a changelog composer writes prose from;
	// they are carried on the record because the blob is read once here, and a
	// later reader could disagree with this one about what a record says.
	Summary string
	// ShippedIn is the release that actually carried this record's work, when the
	// record says so — the `shipped_in:` frontmatter field, parsed as a version.
	// Empty when the field is absent, null, or does not parse.
	//
	// THIS IS A MIGRATION MECHANISM. A repository abcd manages from its first
	// commit should never set it: RS001 makes resolution ride the fixing commit,
	// so a record reaching a terminal folder and the work shipping are the same
	// event, and the cut is right by construction. The field exists because abcd
	// was built before that rule existed — a 2026-08-24 hygiene sweep closed 33
	// records for work released as far back as v0.2.0, and the v0.6.4 cut derived
	// its version from four of them.
	//
	// Say that here rather than in a note elsewhere: a migration accommodation
	// that does not name itself becomes permanent architecture by default, and
	// the next author builds on it (less-but-better).
	//
	// It exists because the cut is a set-difference of FOLDER MEMBERSHIP, which
	// answers "did this record reach a terminal folder since the anchor?" and is
	// read as "did this work ship in this release?". Those coincide only while
	// resolution lands in the fixing commit. A ledger hygiene sweep is the
	// legitimate exception — it closes records for work released long ago — and
	// on 2026-08-24 one closed 33 of them, after which the v0.6.4 cut tried to
	// announce fixes first released in v0.2.0 as new (iss-2608241612087533).
	//
	// Nothing is INFERRED here. An earlier design tested whether
	// resolved_by.commit was reachable from the anchor, which would have judged
	// 85 of 376 resolved records and guessed at the rest. A record either states
	// which release carried it or it does not, and a record that does not say is
	// treated as this cut's — the direction that risks a redundant line rather
	// than a silent omission.
	ShippedIn string
	// ShippedInErr is why a present `shipped_in:` did not parse, empty otherwise.
	// A record carrying an unparseable value stays IN the cut: excluding on a
	// value nobody could read would drop real content from a release record on a
	// typo, which is the worse of the two failures.
	ShippedInErr string
	// PressRelease is the body of the record's `## Press Release` section, empty
	// when it has none. It is the source a release page's quotes are verified
	// against, read from the same blob as Title and Summary so the page and the
	// changelog cannot disagree about what a record says.
	PressRelease string
}

// RecordSet is a release cut: the set-difference of record END STATES between
// the anchor tag and HEAD.
//
// It is deliberately NOT a git-log walk of moves. A squash or rebase merge
// collapses the commit that moved a record into its terminal folder, so a walk
// over `<tag>..HEAD -- <paths>` reports a different set depending on how a
// branch happened to land — the same tree, two answers. Comparing end states
// asks the only question that matters ("which records are in the terminal
// folders now, and which were there at the tag?") and is immune to the shape of
// the history in between.
type RecordSet struct {
	// BaseRef is the anchor the cut is measured from (a tag such as v0.3.0).
	BaseRef string
	// Added are records present at HEAD and absent at the anchor — the cut.
	Added []Record
	// Removed are records present at the anchor and absent at HEAD. A record
	// leaves a terminal folder when it is superseded or re-slugged; either way
	// it is a user-visible change that must surface as a Removed/Changed line
	// rather than silently vanish from the release.
	Removed []Record
}

// All returns the whole cut — added then removed — for callers that must account
// for every record, such as the changelog completeness bijection.
func (s RecordSet) All() []Record {
	out := make([]Record, 0, len(s.Added)+len(s.Removed))
	out = append(out, s.Added...)
	out = append(out, s.Removed...)
	return out
}

// ChangelogRequired is the cut minus every internal record: exactly the set of
// records the generated prose must cite, no more and no less. internal records
// are excluded here (not filtered later, per-caller) so the changelog and the
// version agree on one definition of "user-facing".
func (s RecordSet) ChangelogRequired() []Record {
	out := make([]Record, 0, len(s.Added)+len(s.Removed))
	for _, r := range s.All() {
		if r.InChangelog() {
			out = append(out, r)
		}
	}
	return out
}

// PressReleaseRequired is the release page's set: the ADDED intents whose impact
// earns a changelog line.
//
//	PressReleaseRequired = Added ∩ itd-* ∩ InChangelog
//
// Removed records, issues and `impact: internal` intents fall out by
// construction, and a record carrying a valid `shipped_in:` is already absent
// from Added. An unlabelled Added record never reaches a ready cut
// (UnlabelledAdded refuses it), so InChangelog means "user-facing" here without
// qualification. It is written once, here, so the preview, the composer's view
// of the cut and the page bijection read one definition.
func (s RecordSet) PressReleaseRequired() []Record {
	var out []Record
	for _, r := range s.Added {
		if strings.HasPrefix(r.ID, "itd-") && r.InChangelog() {
			out = append(out, r)
		}
	}
	return out
}

// InChangelog reports whether this record must be cited in the generated prose.
// It is the one definition of "must be reported", written here rather than at
// each call site so the preview, the composer's view of the cut, and the
// completeness bijection can never disagree about which records are required.
//
// It is NOT simply Impact.InChangelog(). An UNLABELLED record is required too,
// and that is the whole reason this method exists: an unlabelled record's
// judgement is unknown, not internal, and the only unlabelled records that ever
// reach a ready cut are on the Removed side (UnlabelledAdded refuses the other
// direction), read from the anchor tag's IMMUTABLE tree. Folding one into "earns
// no line" would drop a supersession from the permanent release record for a
// defect the operator cannot fix — the silent omission UnlabelledAdded's contract
// promises never happens.
func (r Record) InChangelog() bool { return r.Impact.InChangelog() || r.ImpactErr != "" }

// Unlabelled returns every record in the cut whose impact is absent or invalid,
// for a preview that must show the operator the whole picture. It is NOT the
// refusal set — see UnlabelledAdded for that.
func (s RecordSet) Unlabelled() []Record { return unlabelled(s.All()) }

// UnlabelledAdded returns the unlabelled records on the ADDED side only: the
// ones a caller may refuse the cut over, because an unlabelled record ranks
// below every real impact and deriving over it would quietly under-bump a
// release that may contain a break.
//
// Removed records are deliberately excluded. Their blob is read from the anchor
// tag's immutable tree, so one that predates the impact back-fill can never be
// labelled: at HEAD the file is either gone (supersession) or already carries a
// valid impact under its new name (re-slug). Refusing over it would name a
// remedy the operator cannot perform and would block every release until the
// move was reverted. Such a record still travels in Removed, so the release
// still reports it rather than dropping it silently.
func (s RecordSet) UnlabelledAdded() []Record { return unlabelled(s.Added) }

// unlabelled is the one filter both accessors share, so the definition of
// "unlabelled" is written once.
func unlabelled(records []Record) []Record {
	var out []Record
	for _, r := range records {
		if r.ImpactErr != "" {
			out = append(out, r)
		}
	}
	return out
}

// Impact is the strongest impact in the cut — the one that decides the bump.
// An empty or all-internal cut yields ImpactInternal, which reads at the call
// site as "nothing to release".
func (s RecordSet) Impact() Impact { return maxImpactOf(s.All()) }

// DeclaresBreak reports whether the cut declares that it narrows the public
// surface on purpose — the release guardrail's "was this break declared?" test.
//
// It judges the ADDED side only, and that restriction is the whole point of the
// method existing separately from Impact(). A Removed record's blob is read from
// the anchor tag's immutable tree, so its label is what the LAST release
// declared, not what this one does. Supersession is a supported flow — a shipped
// intent leaves the folder when a later one replaces it — and the historical
// impact back-fill puts `breaking` labels on old intents, so a cut that ships
// nothing breaking can easily carry a superseded `breaking` record on its Removed
// side. Counting that as a declaration would wave an undeclared removal through
// the guardrail, silently, which is the exact fail-open the guardrail exists to
// prevent. The same reasoning excludes Removed from UnlabelledAdded.
//
// A withdrawal that genuinely narrows the surface therefore ships its own
// superseding record labelled `breaking`.
//
// Impact() deliberately keeps counting both sides: the version must account for
// every record in the cut, because a record leaving a terminal folder is itself a
// user-visible change.
func (s RecordSet) DeclaresBreak() bool { return maxImpactOf(s.Added) == ImpactBreaking }

// maxImpactOf is the one place a slice of records is reduced to its strongest
// impact, so the version arithmetic and the guardrail can never disagree about
// what the maximum means.
func maxImpactOf(records []Record) Impact {
	impacts := make([]Impact, 0, len(records))
	for _, r := range records {
		impacts = append(impacts, r.Impact)
	}
	return MaxImpact(impacts)
}

// ShippedSince computes the release cut between baseRef (the anchor tag) and
// HEAD, as the set-difference of the record files in the terminal folders.
//
// Both sides are read out of git, never out of the working tree: a removed
// record exists only in the anchor's tree, and reading the added side from git
// too keeps a dirty or half-staged working tree from changing what a release
// reports. Every returned slice is sorted by path — the set feeds a rendered
// preview and a changelog bijection, both of which must be reproducible.
//
// A baseRef git cannot resolve is an error, never an empty set: reporting
// "nothing shipped" against a tag that does not exist would let a cut silently
// claim there is nothing to release.
func ShippedSince(root string, baseRef string) (RecordSet, error) {
	set := RecordSet{BaseRef: baseRef}

	basePaths, err := recordPathsAt(root, baseRef)
	if err != nil {
		return RecordSet{}, err
	}
	headPaths, err := recordPathsAt(root, "HEAD")
	if err != nil {
		return RecordSet{}, err
	}

	// ADDED only. A record that names the release which carried it is not part of
	// this cut: folder membership says it arrived since the anchor, the record says
	// the work shipped earlier, and the record is the more specific claim.
	for p := range headPaths {
		if _, atBase := basePaths[p]; !atBase {
			if rec := newRecord(root, "HEAD", baseRef, p); rec.ShippedIn == "" {
				set.Added = append(set.Added, rec)
			}
		}
	}
	// REMOVED keeps everything, deliberately. `shipped_in` answers "when did this
	// WORK ship", and a record leaving a terminal folder is a different event: the
	// withdrawal or supersession happened in THIS window whenever the work first
	// shipped. Filtering here dropped a supersession's impact from the version
	// arithmetic and, alone in a window, emptied the cut and refused the release —
	// while ingest.go states plainly that letting a removal go uncited "would be
	// the same omission as dropping an addition".
	for p := range basePaths {
		if _, atHead := headPaths[p]; !atHead {
			set.Removed = append(set.Removed, newRecord(root, baseRef, baseRef, p))
		}
	}
	sortRecords(set.Added)
	sortRecords(set.Removed)
	return set, nil
}

// recordPathsAt lists the record files present in the terminal folders at ref,
// keyed by repo-relative path. Non-record files (READMEs, notes) are dropped by
// the shared walk, so every later stage sees records only.
func recordPathsAt(root string, ref string) (map[string]struct{}, error) {
	return recordPathsIn(root, ref, recordPaths...)
}

// maxRecordBytes caps the guarded blob read, in the same order as
// maxChangelogBytes and for the same reason: a record is a page of prose, so a
// file that is not one must not stream unbounded input into a read-only preview.
// Only the frontmatter is read, and that is at the top of the blob, so a
// truncated tail cannot change the parsed impact.
const maxRecordBytes = 4 << 20

// newRecord reads one record's blob at ref and parses its impact. A blob that
// cannot be read, or an impact that cannot be parsed, yields an unlabelled
// record carrying the reason rather than an error: one malformed record must not
// hide the rest of the cut from the operator who has to fix it.
func newRecord(root, ref, baseRef, relPath string) Record {
	rec := Record{Path: relPath, ID: recordID(relPath)}
	blob, err := gitutil.RunLimited(root, maxRecordBytes, "cat-file", "blob", ref+":"+relPath)
	if err != nil {
		rec.ImpactErr = fmt.Sprintf("reading %s at %s: %v", relPath, ref, err)
		rec.Title = rec.ID
		return rec
	}
	// The source material is extracted even when the impact is unlabelled: the
	// operator who has to fix that record is helped by seeing what it says.
	rec.Title, rec.Summary = summarise(blob, rec.ID)
	rec.PressRelease = pressReleaseSection(blob)
	fields := frontmatter.Fields(strings.Split(blob, "\n"))
	// Read BEFORE the impact early-return. A record whose impact does not parse is
	// exactly the legacy population this field exists for — an old record closed by
	// a hygiene sweep — and returning early left it unable to declare itself out of
	// the cut, so it stayed and refused the release as unlabelled with no way to
	// honour the exclusion.
	rec.ShippedIn, rec.ShippedInErr = parseShippedIn(root, baseRef, fields["shipped_in"].Value)
	// A YAML null is an ABSENT impact, not a malformed one. ParseImpact is a value
	// validator and knows only the empty string as absent, so the null spellings
	// ("~", "null", "Null", "NULL") reached it as values and came back as "invalid
	// impact" — while record-lint, which gates on frontmatter.IsNull first, called
	// the same line "impact must be set explicitly". Two diagnoses for one field,
	// one of them telling the operator to fix a typo in a field that has no value
	// at all. Normalising here routes every null to the one missing-impact message
	// (iss-286). frontmatter.IsNull is the single null predicate; no local test.
	raw := fields["impact"].Value
	if frontmatter.IsNull(raw) {
		raw = ""
	}
	impact, err := ParseImpact(raw)
	if err != nil {
		rec.ImpactErr = err.Error()
		return rec
	}
	rec.Impact = impact
	return rec
}

// parseShippedIn reads the `shipped_in:` frontmatter value and decides whether it
// may remove this record from the cut. An absent or null field is neither a value
// nor an error: most records have nothing to say here.
//
// Shape is checked, and then so is TRUTH. A value must name a tag that actually
// exists and that the anchor can reach — meaning a release already published at
// or before the one being measured from. Shape alone is not enough, and the
// distinction is not academic: `v0.64.0` mistyped for `v0.6.4` matches the
// pattern perfectly, names no release, and under a shape-only check removed its
// record from the cut with no error at all. An adversarial review demonstrated
// exactly that with `v9.9.9` on a breaking record, which vanished and left the
// cut empty.
//
// Every rejection keeps the record IN the cut and says why. The field's whole job
// is to take records OUT, so any doubt must resolve toward a redundant changelog
// line rather than a silent omission.
func parseShippedIn(root, baseRef, raw string) (value string, parseErr string) {
	v := strings.TrimSpace(raw)
	if frontmatter.IsNull(v) {
		return "", ""
	}
	if !shippedInRe.MatchString(v) {
		return "", fmt.Sprintf("shipped_in %q is not a release tag (want vMAJOR.MINOR.PATCH)", v)
	}
	if _, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", v+"^{commit}"); err != nil {
		return "", fmt.Sprintf("shipped_in %q names no tag in this repository", v)
	}
	// Reachable from the anchor means "already released by the time this cut
	// starts". A tag that is newer than the anchor, or on an unrelated line of
	// history, cannot have carried work this cut is measuring.
	if _, err := gitutil.Run(root, "merge-base", "--is-ancestor", v, baseRef); err != nil {
		return "", fmt.Sprintf("shipped_in %q is not an ancestor of %s, so it cannot have carried this work", v, baseRef)
	}
	return v, ""
}

// shippedInRe is the release-tag shape this repository tags with: a leading v
// and three dot-separated numbers. Deliberately strict — the field's whole job
// is to remove a record from a release, so a value that cannot be checked must
// not be able to do it silently.
var shippedInRe = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)

// recordID extracts the id from a record path; the caller has already matched
// the filename, so the empty fallback is unreachable in practice.
func recordID(relPath string) string {
	m := recordFileRe.FindStringSubmatch(path.Base(relPath))
	if m == nil {
		return ""
	}
	return m[1]
}

// sortRecords orders by path, the only field guaranteed unique in a set.
func sortRecords(records []Record) {
	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })
}
