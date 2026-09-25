package lifeboat

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/termsafe"
)

// graveyard.go — shared types, constants, and id helpers for the three-layer
// graveyard (M4, adr-35). It carries ONLY the vocabulary the three layers agree
// on; the deterministic builders (graveyard_archaeology.go,
// graveyard_abandoned.go) and the cite-or-dropped validator
// (graveyard_lessons.go) live in their own files so each layer is developed
// independently.
//
// The discipline the graveyard exists to enforce:
//
//   - Layer 1 (Archaeology, archaeology.json) is Tier-0 git only, deterministic,
//     EVIDENCE ONLY — reverted commits, unmerged branches, deleted paths, removed
//     dependencies, wholesale rewrites. No interpretation.
//   - Layer 2 (Recorded abandonment, abandoned.json) is what the project itself
//     declared dead — superseded ADRs and intents, wontfix issues, an ADR's
//     Alternatives-Considered section, rejected options named in a decision log.
//   - Layer 3 (Interpretation, lessons.json) is host-delegated: an agent reads
//     layers 1 and 2 and says what was tried and why. It CANNOT float free —
//     every entry must cite layer-1/2 ids, and the Go validator drops an entry
//     that cites nothing. That validator, not the model's good intentions, is the
//     difference between a graveyard and a séance.
//
// Both archaeology.json and abandoned.json are ALWAYS written (empty Findings
// slice when a repo declared nothing), so the lifeboat's file set is stable, the
// pinned manifest hash is deterministic, and layer 3 can open both files
// unconditionally. lessons.json is written LATER by `abcd disembark graveyard`
// into an already-packed lifeboat and is deliberately NOT part of
// manifest_sha256 (that hash is pinned at pack time over the deterministic
// extraction; interpretation is a separate, mutable layer whose integrity is the
// per-entry cite-or-dropped rule, not the manifest seal).

// GraveyardSchemaVersion stamps archaeology.json and abandoned.json.
const GraveyardSchemaVersion = 1

// LessonsSchemaVersion stamps lessons.json and each low-confidence file.
const LessonsSchemaVersion = 1

// Signal names the specific graveyard signal one Finding reports. It is the
// discriminator layer 3 (and the human renderer) reads to group findings, and
// its fixed ordering (signalRank) gives both graveyard files a deterministic,
// byte-stable layout across re-plans of an unchanged repo.
type Signal string

const (
	// --- Layer 1: archaeology (Tier-0 git, deterministic, evidence only) ---

	// SignalRevert is a commit that reverts an earlier one (a deliberate,
	// explicit abandonment written into history).
	SignalRevert Signal = "revert"
	// SignalUnmergedBranch is a local branch never merged into the default
	// branch, ranked by how long ago it diverged.
	SignalUnmergedBranch Signal = "unmerged-branch"
	// SignalDeletedPath is a path deleted after substantial history — sustained
	// investment abandoned, not a scratch file swept.
	SignalDeletedPath Signal = "deleted-path"
	// SignalRemovedDependency is a dependency (or a whole manifest) present in
	// history but absent at HEAD — a dependency adopted then dropped.
	SignalRemovedDependency Signal = "removed-dependency"
	// SignalWholesaleRewrite is a single commit that replaces a large fraction of
	// the tree — a restructure/rewrite rather than incremental work.
	SignalWholesaleRewrite Signal = "wholesale-rewrite"

	// --- Layer 2: recorded abandonment (Tier-1/2) ---

	// SignalSupersededIntent is an intent in the superseded/ bucket.
	SignalSupersededIntent Signal = "superseded-intent"
	// SignalSupersededADR is an ADR whose status is superseded (or whose
	// superseded_by is non-null).
	SignalSupersededADR Signal = "superseded-adr"
	// SignalAlternativesConsidered is an ADR's "Alternatives Considered" section —
	// options the decision weighed and rejected.
	SignalAlternativesConsidered Signal = "alternatives-considered"
	// SignalWontfixIssue is an issue moved to the wontfix/ ledger bucket.
	SignalWontfixIssue Signal = "wontfix-issue"
	// SignalRejectedOption is a decision-log line naming a rejected option.
	SignalRejectedOption Signal = "rejected-option"
)

// signalRank fixes the order signals appear in a graveyard file. Findings are
// grouped by signal in this order, and within a signal a builder appends in that
// signal's own deterministic order (git-log order, divergence age, sorted path,
// record id, or line number), so the assembled file is byte-identical across
// re-plans of an unchanged repo.
var signalRank = map[Signal]int{
	SignalRevert:                 0,
	SignalUnmergedBranch:         1,
	SignalDeletedPath:            2,
	SignalRemovedDependency:      3,
	SignalWholesaleRewrite:       4,
	SignalSupersededIntent:       5,
	SignalSupersededADR:          6,
	SignalAlternativesConsidered: 7,
	SignalWontfixIssue:           8,
	SignalRejectedOption:         9,
}

// Finding is one graveyard signal, uniform across layers 1 and 2. ID is a stable
// deterministic token (see the id helpers below) that a layer-3 lesson cites in
// its Evidence array; an entry citing no live id is dropped by the validator.
// Summary is a one-line human description.
//
// Two channels carry the rest, and they are kept apart by type rather than by
// wording (iss-2608270926037088):
//
//   - Evidence is the concrete cited material (commit subjects, quoted lines,
//     paths). It is drawn from repository content a hostile or archived repo
//     controls, so every string passes through gvText at build time.
//   - Notices are the binary's own statements ABOUT the finding: a cap omission,
//     a shadowed claimant, a truncated listing. They are authored here, never
//     copied from the repository; where one names a record path, the path is a
//     Go-quoted operand, so it cannot close its quotes and speak as the notice.
//
// When the two shared one array, a crafted path or bullet could write text
// indistinguishable from an omission notice, and the layer-3 interpreter reading
// the file had no way to tell the forgery from the binary's word.
type Finding struct {
	ID       string   `json:"id"`
	Signal   Signal   `json:"signal"`
	Summary  string   `json:"summary"`
	Evidence []string `json:"evidence,omitempty"`
	Notices  []string `json:"notices,omitempty"`
}

// maxGraveyardTextBytes caps one record-drawn graveyard string after cleaning,
// so a single decision line or commit subject cannot balloon a graveyard file.
// It matches the cap one layer-3 lesson's prose takes.
const maxGraveyardTextBytes = maxLessonProseBytes

// gvText cleans one record-drawn string for a graveyard file. Sanitize alone made
// it terminal-safe but left CommonMark and raw-HTML openers live for the layer-3
// interpreter that reads the file; the shared prose cleaner breaks those apart,
// folds the string to one line, and bounds it (iss-2608270926037088).
func gvText(s string) string { return termsafe.CleanProseLine(s, maxGraveyardTextBytes) }

// gvQuoted is a record-drawn operand inside a notice: cleaned, then Go-quoted,
// so the reader sees exactly where the repository's text begins and ends.
func gvQuoted(s string) string { return strconv.Quote(gvText(s)) }

// Archaeology is layer 1: the Tier-0, git-only, deterministic, evidence-only
// dig. Findings is never nil in a written file (an empty [] is a first-class
// result — "history records nothing abandoned").
type Archaeology struct {
	SchemaVersion int       `json:"schema_version"`
	Findings      []Finding `json:"findings"`
}

// Abandoned is layer 2: what the project explicitly declared dead in its record.
// Findings is never nil in a written file (an empty [] honestly states "the
// record declares nothing dead").
type Abandoned struct {
	SchemaVersion int       `json:"schema_version"`
	Findings      []Finding `json:"findings"`
}

// Lesson is one layer-3 interpretation, both the untrusted input shape (the
// host-delegated interpreter emits an array of these) and the written output
// shape. Confidence reuses the coverage Confidence enum (high/medium/low);
// "low" routes the lesson to graveyard/low-confidence/ instead of lessons.json.
// Evidence must cite layer-1/2 Finding ids — an entry with no live ref is
// dropped. Lesson (the prose) is sanitised and marker-neutralised before it is
// written into the lifeboat.
type Lesson struct {
	ID         string     `json:"id"`
	Lesson     string     `json:"lesson"`
	Confidence Confidence `json:"confidence"`
	Evidence   []string   `json:"evidence"`
}

// LessonsFile is the on-disk shape of graveyard/lessons.json (and, for one
// entry, each graveyard/low-confidence/<id>.json).
type LessonsFile struct {
	SchemaVersion int      `json:"schema_version"`
	Lessons       []Lesson `json:"lessons"`
}

// LessonDrop records one lesson the validator refused to write, and why. A drop
// is reported, never fatal: one uncitable entry must not sink an otherwise-good
// interpretation batch.
type LessonDrop struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// LessonsResult is the transport-agnostic outcome of `abcd disembark graveyard`.
// The core returns it; a surface renders it (the core never prints). Written +
// LowConfidence + Dropped accounts for every input entry.
type LessonsResult struct {
	LifeboatDir        string       `json:"lifeboat_dir"`
	Written            int          `json:"written"`
	LowConfidence      int          `json:"low_confidence"`
	Dropped            int          `json:"dropped"`
	Drops              []LessonDrop `json:"drops,omitempty"`
	LessonsPath        string       `json:"lessons_path,omitempty"`
	LowConfidencePaths []string     `json:"low_confidence_paths,omitempty"`
}

// Graveyard tunables. Each is a named constant so its rationale travels with it.
const (
	// substantialHistoryCommits is the number of commits that must have touched a
	// path before its deletion for the deletion to count as abandonment. A path
	// touched by ~10+ commits carried sustained investment; deleting it is a
	// deliberate retirement, not the sweep of a scratch or generated file. Below
	// the threshold a deletion is ordinary churn and is not reported.
	substantialHistoryCommits = 10

	// wholesaleRewriteMinFiles is the absolute floor of files a single commit must
	// change to be a rewrite candidate. It keeps a tiny repo (where one commit can
	// touch "most" of three files) from reading every large commit as a rewrite.
	wholesaleRewriteMinFiles = 25

	// wholesaleRewriteTreeFraction is the fraction of the HEAD tree's file count a
	// single non-merge commit must change to be a wholesale rewrite. Half or more
	// of the tree in one commit is a restructure, not incremental work. HEAD tree
	// size is a deterministic, cheap denominator (an exact per-commit tree size
	// would need a tree walk per commit); the proxy is documented and defensible.
	wholesaleRewriteTreeFraction = 0.5

	// maxGraveyardFindingsPerSignal bounds each signal's findings so a pathological
	// or hostile history cannot balloon a graveyard file. Excess is dropped; the
	// last retained finding for a truncated signal notes the cap in its Notices.
	maxGraveyardFindingsPerSignal = 500

	// maxDependencyTokens bounds the removed-dependency names cited per manifest,
	// so a manifest rewritten wholesale does not list thousands of tokens.
	maxDependencyTokens = 64

	// maxGraveyardCandidatePaths bounds how many deleted paths the deleted-path
	// signal even considers, as defence in depth against a hostile history with a
	// huge number of deletions. The candidate list is sorted, so the bound is
	// deterministic; findings are further capped at maxGraveyardFindingsPerSignal.
	maxGraveyardCandidatePaths = 50000
)

// Lessons (layer-3) trust-boundary caps. The lessons JSON is untrusted host /
// model output, read behind the same guards as an intent verdict.
const (
	// maxLessonsBytes caps the untrusted lessons payload.
	maxLessonsBytes = 1 << 20 // 1 MiB
	// maxGraveyardFileBytes bounds one packed archaeology/abandoned file when the
	// lessons validator reads it back out of a lifeboat.
	maxGraveyardFileBytes = 8 << 20 // 8 MiB
	// maxLessons caps how many entries one ingest may carry.
	maxLessons = 1000
	// maxLessonEvidenceRefs caps the evidence refs read per lesson.
	maxLessonEvidenceRefs = 128
	// maxLessonProseBytes caps one lesson's prose after sanitisation.
	maxLessonProseBytes = 4096
)

// maxLessonIDLen bounds a lesson id's length (the path-traversal defence pairs
// the regex shape with this ceiling before any low-confidence filename is built).
const maxLessonIDLen = 64

// MaxLessonsBytes is the exported cap a front door uses to bound its read of the
// untrusted lessons payload (file or stdin) before handing it to IngestLessons,
// which enforces the same ceiling internally.
const MaxLessonsBytes = maxLessonsBytes

// lessonIDRe constrains a lesson id so it can never build a path that escapes
// graveyard/low-confidence/: "les-" then kebab-case [a-z0-9] segments. The
// validator also enforces maxLessonIDLen before joining any path.
var lessonIDRe = regexp.MustCompile(`^les-[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ---------------------------------------------------------------------------
// Id grammar. Every graveyard finding id is stable across re-plans of an
// unchanged repo (the byte-identical re-plan invariant) and is namespaced by a
// prefix so layer-1 git ids and layer-2 record ids can never collide.
// ---------------------------------------------------------------------------

// revID keys a reverted commit by the first 12 hex of its own SHA: rev-<12hex>.
func revID(fullSHA string) string { return "rev-" + shortHex(fullSHA) }

// rewriteID keys a wholesale-rewrite commit by its short SHA: rewrite-<12hex>.
func rewriteID(fullSHA string) string { return "rewrite-" + shortHex(fullSHA) }

// branchID keys an unmerged branch by its name: branch-<name>. Git forbids
// control characters in ref names; idClean strips any stray control/space so the
// id is a clean, matchable token.
func branchID(name string) string { return "branch-" + idClean(name) }

// deletedPathID keys a deleted path by the path itself: del-<path>. Paths are
// deterministic and stable; idClean strips any control character.
func deletedPathID(p string) string { return "del-" + idClean(p) }

// dependencyID keys a manifest's removed dependencies by the manifest name:
// dep-<manifest>.
func dependencyID(manifest string) string { return "dep-" + idClean(manifest) }

// adrAltID keys an ADR's Alternatives-Considered section off the ADR's own id:
// <adr-id>-alt (e.g. adr-12-alt).
func adrAltID(adrID string) string { return adrID + "-alt" }

// decisionID keys a rejected-option decision-log line by its 1-based line number
// in DECISIONS.md: dec-L<line>. The line number is stable for an unchanged file
// (the only stability the re-plan invariant requires).
func decisionID(line int) string { return fmt.Sprintf("dec-L%d", line) }

// shortHex returns the first 12 hex characters of a commit SHA, lower-cased and
// filtered to [0-9a-f] so a stray character never leaks into an id.
func shortHex(sha string) string {
	out := make([]byte, 0, 12)
	for i := 0; i < len(sha) && len(out) < 12; i++ {
		c := sha[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
			out = append(out, c)
		case c >= 'A' && c <= 'F':
			out = append(out, c+('a'-'A'))
		}
	}
	return string(out)
}

// idClean makes an id component safe by percent-encoding control characters,
// DEL, spaces, and the escape byte '%' itself. It must be INJECTIVE: the old
// version simply DELETED spaces/controls, so distinct inputs ("my file" and
// "myfile", or two branch names differing only in whitespace) collapsed to the
// same id and one finding silently shadowed the other. Percent-encoding keeps the
// token compact and space-free while guaranteeing distinct inputs yield distinct
// ids. Ordinary paths and ref names (no control/space/'%') pass through
// unchanged, so their ids stay stable across the re-plan invariant.
func idClean(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f || r == ' ' || r == '%' {
			fmt.Fprintf(&b, "%%%02X", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// Finding caps. ONE capping primitive, shared by both layers: every signal that
// drops findings at maxGraveyardFindingsPerSignal announces the drop on the last
// retained finding, so a packed graveyard file never presents a truncated signal
// as complete. Layer 2 used to carry its own silent variant, which broke the
// contract stated on the constant for six of the ten signals.
// ---------------------------------------------------------------------------

// capSignalFindings bounds one signal's findings at maxGraveyardFindingsPerSignal
// so a pathological or hostile history cannot balloon a graveyard file. When it
// truncates, the last retained finding notes the cap, so the file honestly
// declares that more was dropped rather than silently hiding it.
func capSignalFindings(fs []Finding) []Finding {
	if len(fs) <= maxGraveyardFindingsPerSignal {
		return fs
	}
	return noteFindingsOmitted(fs[:maxGraveyardFindingsPerSignal], len(fs)-maxGraveyardFindingsPerSignal)
}

// noteFindingsOmitted writes the cap notice onto the last retained finding's
// Notices. It is the announcing half of capSignalFindings, split out for the one
// signal (unmerged-branch) whose list is bounded BEFORE its findings are built:
// there the count dropped is known only at the probe, so a findings-level cap can
// never see it. The notices slice is copied before the append because fs may
// share its array with the caller's untruncated slice. A drop with no retained
// finding to carry it has nowhere to be announced and leaves fs untouched.
func noteFindingsOmitted(fs []Finding, omitted int) []Finding {
	if omitted <= 0 || len(fs) == 0 {
		return fs
	}
	last := &fs[len(fs)-1]
	last.Notices = append(append([]string(nil), last.Notices...),
		fmt.Sprintf("+%d further findings omitted; capped at %d", omitted, maxGraveyardFindingsPerSignal))
	return fs
}

// collectFindingIDs builds the set of live finding ids from any number of finding
// groups (layer 1 + layer 2), the membership set a layer-3 lesson's evidence must
// hit to be written.
func collectFindingIDs(groups ...[]Finding) map[string]bool {
	ids := map[string]bool{}
	for _, g := range groups {
		for _, f := range g {
			if f.ID != "" {
				ids[f.ID] = true
			}
		}
	}
	return ids
}
