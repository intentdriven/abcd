---
id: itd-42
slug: coherence-aware-grill
spec_id: spc-2609211918551301
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: []
prd_path: null
grilled_at: 2026-05-16T17:21:40Z
grill_session_id: 1d0ed7a5-b52e-455d-b353-4b30685ad7ae
grilled_intent_hash: f56308389573c7a4c993da27eeb0243365843713b27dc2c3532b7515ae9d9247
glossary_terms_used:
  - core/brief
  - core/intent
  - core/oracle
  - core/phase
  - interview/session
warrants_assumed:
  - Product thinkers act on a surfaced sibling-overlap — they kill or merge the redundant intent rather than answering "they're different" and moving on. The grill captures the answer (per acceptance criteria) but does not enforce that the answer is honest.
blocked_by: [itd-27]
builds_on: [itd-41]
severity: major
impact: additive
---

# Grill Reads an Intent Against the Brief and Its Siblings, Not Just the Glossary

> **Re-scoped on 2026-09-21** by the product thinker: this record is the automated pre-pass the decomposition discipline (itd-84) names as its next rung. Before the planning interview, abcd reads the brief's invariants, the principles and a one-line index of every intent, and writes the coherence questions into the planning brief the interview starts from. The grill it first named is superseded (itd-27); the press release and scope below are read through this paragraph and the Decisions section.


## Press Release

> **abcd's grill stops checking an intent in isolation: a full grill now reads it against the brief's invariants, its scope boundary, and every other intent — and asks the coherence questions a solo interrogation cannot.** Capturing an idea stays one line and zero friction. But when a product thinker promotes a draft, the grill loads more than the terminology glossary: it loads the brief's invariants and scope sections, and a one-line index of every other intent — drafted, planned, and shipped. Now it can ask the question that actually catches wrong code: "Invariant 3 says config is never written outside `~/.abcd/` — your intent implies it is; which gives?" and "itd-19 already covers stage-aware behaviour — how is this different?" It still grills vague terms and hidden assumptions; it now also grills *incoherence* — the intent that is locally clear and globally wrong. Grilling against the corpus is grounded the way the phase negotiator is grounded: a coherence concern that cannot be tied to a named invariant, scope clause, or sibling intent is asked as a Socratic question, never asserted as a conflict.
>
> "I'd grill an intent and it would come out crisp — clear terms, testable acceptance — and still be quietly redundant with something I'd specced two months earlier," said Iris, product lead. "abcd asked me, straight out: 'itd-31 already does cross-document fidelity — what does this add?' It didn't decide for me. It made me say the difference out loud, or admit there wasn't one. I killed one intent and sharpened another, before either reached a spec."

## Why This Matters

The intent layer is abcd's highest-leverage moment for product clarity ([itd-27](../superseded/itd-27-grill-skill-and-glossary.md) built `/abcd:intent grill` on exactly that premise). But the grill itd-27 shipped has a blind spot its `--with-docs` flag name actively hides: "glossary-aware" mode loads **only** the terminology database. It checks that an intent *speaks* the brief's words. It never reads the brief, never reads another intent, never reads a shipped spec. It enforces vocabulary; it does not enforce coherence. This intent also corrects the misnomer: `--with-docs` becomes `--glossary` (the terminology tier it always was), the new coherence tier is `--coherence`, and `--full` runs both.

So an intent can pass a full grill — crisp terms, EARS-clean acceptance, warrants surfaced — and still:

- violate a `02-constraints/03-invariants.md` invariant nobody re-read at intent time;
- propose work the brief's `01-product/04-scope.md` or `06-delivery/03-out-of-scope.md` already rules out;
- duplicate or contradict a sibling intent (the corpus already has 40+).

Each of those is caught today only later — at planning, at review, or not at all — when the fix is a re-plan instead of a three-minute conversation. The intent stage is where coherence is cheapest to enforce, and the current grill does not enforce it.

abcd already has the grounded-adversary pattern this needs. [itd-41](../drafts/itd-41-phase-negotiator.md)'s phase negotiator is *Socratic where it questions, grounded where it asserts* — it never invents a trade-off to sound thorough. Coherence grilling MUST work the same way: a hallucinated conflict ("this contradicts itd-12") spends the product thinker's trust on a fiction. Every asserted conflict cites a real anchor; every concern that cannot be anchored is a question.

The brief is already structured for selective loading — numbered sections, invariants and scope already isolated in their own files — so reading the *relevant* slice of it is a context-selection problem, not a new subsystem. Whole-corpus full-text comparison across all intents is not; that is left out of scope and deferred to scope-aware retrieval ([itd-39](../drafts/itd-39-scope-aware-memory-retrieval.md)).

## What's In Scope

- **A coherence tier in the grill, distinct from the glossary tier.** The glossary tier (forbidden-synonym, cross-context, emerging-term, ADR offers — all of itd-27) is unchanged in behaviour. This intent adds a *second, separate* context tier rather than overloading the existing one, because vocabulary checking is cheap and deterministic while coherence checking is expensive and judgement-bound. The flags are renamed to name the tiers honestly: `--with-docs` → `--glossary`, new `--coherence`, and `--full` = both. (This renames the flag itd-27 shipped; itd-27's surface table and the grill `SKILL.md` flag list are updated accordingly — see References.)
- **Lifecycle-defaulted tier selection.** The tier set is *derived* from the intent's location: an intent in `drafts/` defaults to a *light* grill (clarity only — the existing behaviour, glossary optional); an intent being **promoted out of `drafts/`** defaults to the *full* grill: glossary + brief-coherence + sibling-coherence. Explicit `--light` / `--full` flags override the default for the rare case (forcing a full grill on a draft to think it through early). This makes the full grill the default at promotion (the existing `GR002` no-PRD gate) without taxing capture.
- **Brief-coherence context (Tier 2).** A full grill always loads `02-constraints/03-invariants.md`, `02-constraints/04-naming.md`, `01-product/04-scope.md`, `06-delivery/03-out-of-scope.md`, and the `principles/` set (one small file per cross-cutting principle); and loads the matching `04-surfaces/0N-*.md` when the intent names a surface. It does not load the whole brief. **A Tier 2 file that is missing (the brief is mid-migration; paths can move) is skipped with a warning recorded in the grill report — the grill continues with whatever loaded, degrading toward pure Socratic questioning rather than aborting** (the itd-41 degradation limit).
- **Sibling-coherence context (Tier 3), index-level.** A full grill builds, fresh on each run, a one-line-per-intent index — ID, slug, the intent's opening headline sentence, lifecycle state — by scanning `drafts/`, `planned/`, and `shipped/`. No maintained index file: built each grill, so it is never stale. The one-line rule is uniform across all three directories; for `shipped/` intents the line still describes the *idea*, not delivered reality (delivered-vs-intent drift is `intent-fidelity-reviewer`'s job, not the grill's). Index-level only: enough to ask "how does this differ from itd-N?", not full-body semantic comparison.
- **Grounded coherence assertions.** Where the grill asserts a conflict it MUST cite a *stable, authored* anchor — a named invariant or a named scope clause. Sibling-intent overlap is **never asserted**: draft intents are mutable, so a sibling ID is a moving target, and overlap is always surfaced as a Socratic question naming the sibling. A concern with no stable anchor is likewise raised as a Socratic question tagged with a named move — the same taxonomy itd-27's grill already uses — never as an asserted conflict.
- **Coherence findings in the existing artefacts.** Coherence questions and grounded conflicts are recorded in the Phase 1 `grill-report.{json,md}` alongside the existing question stream; the Phase 2 PRD reflects resolved coherence concerns in its sections. No new artefact type.

## What's Out of Scope

- **Replacing or rewriting the glossary tier** — itd-27's glossary-aware mode is kept verbatim. This intent is additive.
- **Full-body semantic comparison across all intents** — Tier 3 is index-level only. Loading and comparing full intent bodies at scale is the scope-aware retrieval problem; deferred to [itd-39](../drafts/itd-39-scope-aware-memory-retrieval.md). This intent must not pre-build that subsystem.
- **Loading the whole brief** — only the named invariant / scope / surface slices. Whole-brief grilling stays out of scope for the same token-budget reason itd-27 ruled out whole-brief grilling.
- **Asserting unanchored conflicts** — a coherence concern that cannot be tied to a named invariant, scope clause, or sibling intent is a question, never an assertion. "Sounds thorough" is not a licence to invent a conflict.
- **Auto-resolving conflicts** — the grill surfaces incoherence; it does not edit the intent, kill a sibling, or rewrite scope. The product thinker decides.
- **Grilling against shipped *code*** — Tier 3 reads shipped *intents*, not the implementation. Delivered-reality comparison remains `intent-fidelity-reviewer`'s job at the shipped transition.
- **A new sub-verb or command** — this is a capability of the existing `/abcd:intent grill`, not a sibling verb.

## Mechanism

We expect a pre-pass that reads the invariants and the sibling index to catch the contradiction or the duplicate before a spec exists, because both are visible from the record alone and the interview today finds them only when the person happens to remember; shown wrong if planned intents still turn out to duplicate or contradict one another after it ships.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a draft intent, **when** the pre-pass runs, **then** the planning brief it writes names every brief invariant the draft's text implies a conflict with, quoting the invariant's line and the draft's.
- **Given** a draft that overlaps an existing intent on any shelf, **when** the pre-pass runs, **then** the brief names the sibling from a one-line index of every intent, writes the question with the four answers (keep both, bundle, supersede, refine), and, where it has one, its recommendation with the reason in the prose beside the question and never as a marked option.
- **Given** the pre-pass has run, **when** the tree is inspected, **then** the draft, the brief and the sibling intents are unchanged; the pre-pass read the invariants, the principles, the index and the draft, and wrote only the planning brief under the local tier.
- **Given** a planning brief with questions, **when** the interview runs, **then** each question is asked, and the answer lands on the record as a decision or a typed link.
- **Given** a concern the pre-pass cannot anchor to a named invariant or record, **when** it writes the brief, **then** the concern is a question marked unanchored, not a finding.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **The record is the pre-pass**, run before the interview and writing into the planning brief; the interview stays the human's.
2. **Overlaps are asked with the four standard answers**, and the pre-pass may recommend one with its reason in the prose beside the question, never as a marked option (the GRILL rule).
3. **The loader is the interview's own** (the planning-brief writer the intent page describes), not a module shared with the phase negotiator or the fidelity reviewer; the context is the invariants, the principles, the index and the draft, and nothing else, which is the budget.

## Open Questions

_None open; decisions 2 and 3 settle the three this record carried._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-e6bcb27c8050 -->
Fidelity review — receipt rcp-e6bcb27c8050 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:88aaf52aee45e546cd9e96b2caf7fce6794983db71693f6234af904a86cbd2c4
Input attestations: diff:85d0bb8eb..8586edba0 (PR #758 feat/intent-prepass, judged at main 7fb52a6b5)@sha256:7803ef19ee07b3c1d7434610d56d0896cf83557ccee5b610ec78b289bee3ed4b;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: Every conflict the host returns is written only when its anchor exists and both quotes occur verbatim (anchor's text and draft's), and the brief prints both quotes under the invariant's number and title; the real register parses to 20 numbered invariants. Concern: which invariants the draft conflicts with, and that EVERY one is named, is the host's judgement (adr-25, spec scope 2) — the binary guarantees anchoring and quoting, not completeness, and no test exercises completeness.
  evidence: internal/core/intent/prepass.go:550-590 — "why = prepassQuoteMiss(c.AnchorQuote, text, prepassAnchorName(c)) ... why = prepassQuoteMiss(c.DraftQuote, in.Draft, "the draft")"
  evidence: internal/core/intent/prepass.go:761-766 — "fmt.Fprintf(&b, "%s says:\n\n> %s\n\n", prepassLine(qt[0]), prepassBlock(qt[1], prepassQuoteCap))"
  evidence: internal/core/intent/prepass.go:249-287 — "func parsePrepassInvariants(content string) []PrepassInvariant"
  evidence: internal/core/intent/prepass_test.go:229 — "func TestPrepassBriefQuotesEveryInvariantConflict"
  evidence: .abcd/development/specs/closed/spc-2609211918551301-coherence-aware-grill.md:22-25 — "a host-delegated judgement (adr-25) over the draft against each invariant"
- ac-2 — MET: The index is built fresh from every intent the corpus loader lists across all five shelves (drafts, planned, shipped, disciplines, superseded), an overlap must name a sibling in it or is demoted, the brief lists the four answers as plain bullets, and a recommendation is rendered as a prose sentence after the list, never as a marked option; the test asserts the shape.
  evidence: internal/core/intent/prepass.go:220-233 — "for _, sib := range corpus.Intents { ... in.Index = append(in.Index, PrepassIndexEntry{ID: sib.ID, Title: title, Shelf: sib.Bucket})"
  evidence: internal/core/intent/intent.go:54 — "var Buckets = []string{BucketDrafts, BucketPlanned, BucketShipped, BucketDisciplines, BucketSuperseded}"
  evidence: internal/core/intent/prepass.go:592-618 — "sib, ok := prepassSibling(in, o.Sibling)"
  evidence: internal/core/intent/prepass.go:768-779 — "fmt.Fprintf(&b, "The pre-pass leans towards **%s**: %s\n\n""
  evidence: internal/core/intent/prepass_test.go:269 — "func TestPrepassOverlapIsAskedWithTheFourAnswersAndTheRecommendationInProse"
- ac-3 — MET: AssemblePrepass reads the draft, the invariants register, the principles directory and every sibling (to build the index) and writes nothing; WritePrepassBrief writes one file, the planning brief under the local tier, through an os.Root, and never replaces a brief the pre-pass did not write. The core test diffs the whole tree before and after and finds only the brief added; a run on the real tree against itd-10 left git status empty.
  evidence: internal/core/intent/prepass.go:171-236 — "func AssemblePrepass(repoRoot, intentID string) (PrepassInput, error)"
  evidence: internal/core/intent/prepass.go:448-459 — "rel := PlanningBriefsRelDir + "/" + in.Intent + ".md" ... fsutil.WriteFileAtomicInRoot(root, rel, []byte(brief), 0o644)"
  evidence: internal/core/intent/prepass.go:64 — "const PlanningBriefsRelDir = ".abcd/.work.local/scratch/planning-briefs""
  evidence: internal/core/intent/prepass_test.go:319 — "func TestPrepassWritesOnlyThePlanningBrief"
- ac-4 — MET_WITH_CONCERNS: Every question the brief writes ends in a `Lands as:` line naming its landing (a decision in `## Decisions`, a draft change, or a typed link via `abcd intent reclassify` / `plan --bundle`), and the plugin page's interview opens from the brief, asks each `### Qn` in order and records each answer where its `Lands as:` line says before the next. Concern: the asking and the landing are host instruction on the page (the interview is a human-session act, intent decision 1); the only test reads the page's prose, and nothing mechanical records or checks that an answer landed.
  evidence: internal/core/intent/prepass.go:766 — "Lands as: a decision in the draft's `## Decisions` saying which gives, or a change to the draft's text before it is planned."
  evidence: internal/core/intent/prepass.go:782-786 — "Lands as: keep both, a decision ... supersede, `superseded_by` on the record that gives way (`abcd intent reclassify <itd-N> --kind superseded --by <itd-M> --reason "<why>"`)"
  evidence: commands/intent.md:405-415 — "Record each answer where its `Lands as:` line says before the next question: a decision line in the draft's `## Decisions`, a change to the draft's text, or a typed link through `abcd intent reclassify`."
  evidence: internal/core/intent/prepass_test.go:412 — "func TestPrepassEveryQuestionSaysWhereItsAnswerLands"
  evidence: internal/surface/cli/intent_prepass_cli_test.go:319 — "func TestIntentPageInterviewOpensFromThePrepass"
- ac-5 — MET: A concern the host marks unanchored, a conflict whose anchor is absent or whose quotes do not occur, and an overlap naming no sibling in the index are all written as `### Qn. A question (unanchored)` with a `Not anchored:` reason, and are counted as demoted; the test covers both the host-declared and the demoted paths.
  evidence: internal/core/intent/prepass.go:580-582 — "demoted = append(demoted, prepassQuestion{kind: "unanchored", question: c.Question, why: why})"
  evidence: internal/core/intent/prepass.go:604-611 — "why := "no intent on any shelf is " + o.Sibling"
  evidence: internal/core/intent/prepass.go:787-793 — "### Q%d. A question (unanchored) ... Not anchored: %s, so it is asked, not asserted."
  evidence: internal/core/intent/prepass_test.go:353 — "func TestPrepassUnanchoredConcernIsAQuestionMarkedUnanchored"

Gap audit:
- honoured:
  - The record is the pre-pass, run before the interview, and the planning brief under the local tier is the only file it writes (decision 1)
    evidence: internal/core/intent/prepass.go:420-464 — "func WritePrepassBrief(repoRoot, intentID string, raw []byte)"
  - Overlaps are asked with the four standard answers and the pre-pass's lean is prose beside the question, never a marked option (decision 2)
    evidence: internal/core/intent/prepass.go:770-779 — "for _, a := range PrepassAnswers { fmt.Fprintf(&b, "- **%s**\n", prepassAnswerLabel[a]) }"
  - The loader is the interview's own and reads only the invariants, the principles, the index and the draft (decision 3)
    evidence: internal/core/intent/prepass.go:52-54 — "The loader is the interview's own (decision 3): it is not shared with the phase negotiator or the fidelity reviewer, and it reads nothing beyond the four inputs"
  - Grounded where it asserts, Socratic where it questions: an unanchorable conflict is demoted to a question, never refused and never asserted
    evidence: internal/core/intent/prepass.go:574-582 — "if why != "" { demoted = append(demoted, ...); continue }"
  - The sibling index is built fresh on every run from every shelf, with no maintained index file
    evidence: internal/core/intent/prepass.go:220-233 — "for _, sib := range corpus.Intents"
  - A missing invariants register or principles directory degrades the pass with a warning in the brief rather than aborting
    evidence: internal/core/intent/prepass.go:202-219 — "case errors.Is(err, fs.ErrNotExist): in.Warnings = append(in.Warnings, ..."
    evidence: internal/core/intent/prepass_test.go:516 — "func TestPrepassMissingInputsDegradeWithAWarning"
  - Wired on both front doors: `abcd intent prepass` on the CLI and the opening of the plugin page's planning interview, plus the Autonomous runs paragraph
    evidence: internal/surface/cli/intent_prepass.go:25 — "Use: "prepass < itd-N> [--findings-json < path>]""
    evidence: commands/intent.md:377-381 — "**Opening: the pre-pass (itd-42).** ... Run `"${CLAUDE_PLUGIN_ROOT}/abcd" intent prepass <itd-N> --json`"
    evidence: commands/intent.md:671-676 — "it runs the pre-pass the interview opens with"
  - The host's findings are untrusted: strict decode, caps, digest match against the input as it stands, and host prose cannot forge the brief's structure
    evidence: internal/core/intent/prepass.go:439-441 — "if f.InputDigest != in.Digest { return ... an input moved since the pass; re-run the pre-pass"
    evidence: internal/core/intent/prepass_test.go:536 — "func TestPrepassHostProseCannotForgeTheBriefStructure"
- diverged:
  - The press release promises a coherence tier inside the grill with `--with-docs` renamed to `--glossary`, new `--coherence`, `--full`, and lifecycle-defaulted tier selection; none of that exists. The delivery is a separate pre-pass verb before the planning interview, as the 2026-09-21 re-scope paragraph signs off (the grill it named is superseded with itd-27)
    evidence: .abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:30 — "**Re-scoped on 2026-09-21** by the product thinker: this record is the automated pre-pass the decomposition discipline (itd-84) names as its next rung"
    evidence: internal/surface/cli/intent_prepass.go:68 — "cmd.Flags().StringVar(&findingsJSON, "findings-json", "", ..."
  - Tier 2 context promised 03-invariants, 04-naming, 01-product/04-scope, 06-delivery/03-out-of-scope and the matching surface chapter; the delivery reads the invariants register and the principles only, as decision 3 narrows it
    evidence: internal/core/intent/prepass.go:66-70 — "invariantsRelPath = ".abcd/development/brief/02-constraints/03-invariants.md" principlesRelDir = ".abcd/development/principles""
    evidence: .abcd/development/intents/shipped/itd-42-coherence-aware-grill.md:92 — "the context is the invariants, the principles, the index and the draft, and nothing else, which is the budget"
  - A conflict can be anchored only to an invariant by number or a principle by path; the scope clauses the press release named as anchors cannot anchor a conflict, so a scope concern is demoted to an unanchored question
    evidence: internal/core/intent/prepass.go:386-392 — "type prepassConflict struct { Invariant *int; Principle string; ..."
    evidence: internal/core/intent/prepass.go:554-555 — "if (c.Invariant == nil) == (c.Principle == "") { return ... names exactly one anchor, an invariant or a principle"
  - The index line promised ID, slug, the opening headline sentence and lifecycle state; delivered is ID, the first H1 title (slug as fallback) and shelf
    evidence: internal/core/intent/prepass.go:127-131 — "type PrepassIndexEntry struct { ID string; Title string; Shelf string }"
    evidence: internal/core/intent/prepass.go:228-231 — "title := prepassTitle(string(content)); if title == "" { title = sib.Slug }"
  - Coherence findings were promised inside grill-report.{json,md} and the Phase 2 PRD; they land in a new planning-brief artefact under the local tier instead
    evidence: internal/core/intent/prepass.go:62-64 — "PlanningBriefsRelDir is where a planning brief lives: the local tier"
- missing:
  - A verified landing for ac-4: nothing in the binary records that an interview question was asked or that its answer reached the record; the asking and recording are host instruction on the plugin page, and the one test reads the page's prose (settled as the human's act by intent decision 1, so recorded here, not captured)
    evidence: commands/intent.md:405-415 — "Record each answer where its `Lands as:` line says before the next question"
    evidence: internal/surface/cli/intent_prepass_cli_test.go:319-354 — "if !strings.Contains(opening, want) { t.Fatalf("the interview's opening, before step 1, does not name %q"
<!-- abcd-review-end receipt=rcp-e6bcb27c8050 -->

## References

- Extends: [itd-27](../superseded/itd-27-grill-skill-and-glossary.md) (grill skill & glossary) — adds a coherence tier to the grill itd-27 built; the glossary tier's behaviour is unchanged. **Also renames itd-27's `--with-docs` flag to `--glossary` and adds `--coherence` / `--full`** — itd-27's surface table and the grill `SKILL.md` flag list must be updated when this intent is planned.
- Shares the grounded-adversary pattern with: [itd-41](../drafts/itd-41-phase-negotiator.md) (phase negotiator) — Socratic where it questions, grounded where it asserts.
- Defers to: [itd-39](../drafts/itd-39-scope-aware-memory-retrieval.md) (scope-aware memory retrieval) — full-body cross-intent comparison at scale is itd-39's problem, not this intent's.
- Coordinates with: [itd-48](itd-48-intent-fidelity-reviewer-roles-2-3.md) (cross-document fidelity reviewer — supersedes [itd-31](../superseded/itd-31-cross-document-fidelity-reviewer.md)) — different register: itd-48's Role 2 reviews delivered documents for drift; this grills an intent for coherence before it is planned.

## Grounds

- pursued: the autonomous run may prepare interviews but not perform them, and the planning brief it writes is only worth reading if this pass fed it; we expect the first briefs to carry a conflict or an overlap the person had not seen; shown wrong if the briefs raise nothing the person did not already know
