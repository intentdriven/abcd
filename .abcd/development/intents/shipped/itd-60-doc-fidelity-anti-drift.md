---
id: itd-60
slug: doc-fidelity-anti-drift
spec_id: spc-2609020903498198
kind: standalone
suggested_kind: standalone
reclassification_history: []
builds_on: [itd-73, itd-80]
related_adrs: []
prd_path: null
grill_session_id: 60d0f1de-0001-4a60-9c0d-000000000060
glossary_terms_used:
- core/brief
- core/intent
- core/spec
- core/oracle
grilled_intent_hash: bfaa672163edddb5d859bbdfc50169512349da38405c295729ce42c04b894401
prd_grandfathered: false
severity: major
impact: additive
---

# When A Surface Ships, The Brief Describes It — Or The Intent Does Not Reach Shipped

## Press Release

> _Sequenced 2026-09-01 as a delivery rung of
> [Phase 8 — the brief is the shipped state](../../roadmap/phases/phase-8-brief-currency.md);
> the phase records the sync rule and its one legitimate lead._


> **abcd gains a doc-fidelity gate: a change cannot call itself shipped while
> the brief lags the surface it delivered, and a release cannot be cut while any
> intent shipped since the last cut leaves its chapter behind.** The gate stands
> at both moments, deliberately. It runs in two layers. The deterministic layer
> needs no oracle and refuses on its own: every verb, every sub-verb and every
> agent the binary ships must have a brief chapter naming it. The semantic layer
> is host-delegated: it reads what the change actually delivered against that
> chapter and refuses on a confirmed false sentence, and where the reviewer
> cannot be reached it fails closed rather than reporting a clean brief. When the
> gate finds the brief lagging, the pass drafts the brief edit **and applies it**
> in the shipping change, then flags it for the product thinker to read — so the
> change is never held hostage to a paragraph, and the paragraph is never left
> unread.

> "I read the verdict, not the source," said a product thinker shipping with
> abcd. "If the brief is six specs behind the code, my whole picture of what I
> have built is wrong, and nothing tells me. I want the framework to refuse to
> call the work shipped until the brief says what is actually true — and to hand
> me the sentence it wrote so I am reviewing, not writing."

## Why This Matters

abcd's external honesty is part of its safety proposition: a product thinker who
cannot read code trusts the brief to tell them what they have built. The brief
is the shipped state (adr-5), and Phase 8 is what makes that decision
enforceable rather than aspirational — a product thinker opens the brief and
knows what abcd does today, without opening the command reference, the changelog
or the code.

Nothing enforces it today. abcd already grades one artefact against another with
discipline: the intent-fidelity reviewer grades delivery against acceptance
criteria, fails closed, and never infers a pass from absent evidence. The same
discipline applied to the brief closes the loop — built reality against the
chapter that claims to describe it. Without it, the framework that most
rigorously grades *delivery against intention* has no guard on *documentation
against delivery*, which is the one surface a non-expert actually reads.

The enforcement point is not arbitrary. `abcd spec close` is the verb that moves
a planned intent to `shipped/`, and it runs in the change that lands the work —
the one moment at which whoever made the change still has the surface in their
head. A gate anywhere later is archaeology.

This is the **forward** direction: built reality drives the brief. The
**reverse** direction — a human editing the brief, with the implied roadmap
changes drawn out — is a separate, paired intent
([[itd-61-brief-change-derivation]]).

## Decisions (grilled 2026-09-01)

The product thinker resolved the open questions at the planning interview; these are
commitments, not options.

- **The gate refuses at both moments.** An intent's move to shipped is refused
  while the brief lags the surface it delivered, and the release cut refuses
  again over every shipped intent. Belt and braces, deliberately.
- **Built reality is checked in two layers.** A deterministic layer refuses on
  its own with no oracle: every verb, sub-verb and agent the binary ships must
  have a brief chapter naming it. A semantic layer, host-delegated, reads what
  the intent delivered against the chapter and refuses on a confirmed false
  sentence; when the reviewer is unavailable the gate fails closed.
- **Brief only, for Phase 8.** A stale public-doc sentence is reported, never
  refuses; the public docs are a later rung.
- **Draft and apply, review after.** When drift is found the pass edits the
  brief in the shipping change and flags it for the product thinker's review; the
  brief may carry a sentence the product thinker did not write until they read it.
- **Sequenced as Phase 8**, the brief is the shipped state, with itd-147 as the
  other rung; standalone kind, its own spec.
- **The gate can run fully autonomously (maintainer, 2026-09-02).** A flag
  (the spec names it) lets an unattended run, an autonomous release cut for
  one, execute both layers without waiting for a person: the deterministic
  layer refuses on its own, the host-run reviewer is invoked by the routine
  the way the changelog composer is, the brief edit is drafted and applied,
  and every applied edit is listed for review after in the change the run
  produces. Autonomy changes who waits, never what refuses: the gate still
  refuses on what it can prove and still fails closed when the reviewer is
  unavailable.

## What's In Scope

- **Two layers, one gate.** The pass is two mechanisms stacked. Layer 1 grades
  *coverage* and needs nothing but the binary and the brief. Layer 2 grades
  *meaning* and is host-delegated. A refusal from either is a refusal.
- **Layer 1 — the deterministic coverage floor.** Every verb, every sub-verb and
  every agent the binary ships must have a brief chapter naming it. The check is
  derived from the command tree and the agent set, calls no oracle, and refuses
  on its own. It is cheap enough to run at every enforcement point and in CI.
- **Layer 2 — the semantic doc-fidelity pass.** Host-delegated review of what the
  change delivered against the chapter that claims to describe it, refusing on a
  **confirmed** false sentence and naming the sentence. Meaning cannot be judged
  deterministically, which is why this layer is delegated rather than coded.
- **Fail closed.** No reviewer, no reliable comparison, no verdict — the gate
  refuses. It never reports a clean brief from absent evidence.
- **Two enforcement points, one mechanism.** `abcd spec close`, the verb that
  moves a planned intent to `shipped/`, refuses per intent while the brief lags
  the surface that intent delivered; the release cut refuses again over every
  intent shipped since the last cut. Belt and braces, deliberately: the per-intent
  point catches the change while its author is present, and the cut catches
  whatever reached `main` around it.
- **Draft and apply, review after.** When the gate finds the brief lagging, the
  pass writes the brief edit into the shipping change itself and flags it for the
  maintainer's review. The brief may carry a sentence the maintainer did not
  write until they read it; that trade is the ruling, and the flag is what makes
  it honest.
- **The per-task pass stays a report.** After a task it reports where the brief
  may lag and blocks nothing. It is an advisory surface, not a gate; the gate is
  the refusal at the two enforcement points above.
- **The one legitimate lead is not drift.** A brief edited after a cut is ahead
  of that release until the next cut, and the gate does not report it as wrong
  for being ahead.

## What's Out of Scope

- **The public docs — a later rung, reported and never refusing.** For this
  phase the gate is **brief only**. A public-doc sentence that lags is reported,
  and the move and the cut proceed regardless. Everything the pre-ruling draft
  claimed here is struck to that later rung: grading the brief against the public
  docs, drafting audience-adapted public-doc deltas, and the end-user /
  developer-extending split. The brief has to hold before anything derived from
  it is worth gating.
- **The docs-currency lint that already shipped from this draft.** `abcd docs
  lint` — change-narration, resolvable cross-links, no stray root documents — is
  decidable without reading the code and runs everywhere already. It is the floor
  this gate stands on, not a layer of it, and this rung neither changes nor
  re-claims it.
- **The reverse direction.** Drawing implied intents and principles out of a
  *human-authored brief edit* is [[itd-61-brief-change-derivation]], a separate
  intent that stays adjacent to this phase.
- **Re-grading delivery against intention.** That is the intent-fidelity
  reviewer's job; this pass consumes "what shipped" as an input and does not
  re-derive it.
- **Authoring the brief's prose voice.** The pass drafts a delta against an
  existing chapter; wholesale authorship and information architecture are not its
  concern.
- **Deciding sequencing or dependencies** relative to the paired intents — that
  is `abcd intent plan`'s job.

## Mechanism

> _Facilitator-seeded from the phase's expectation. The maintainer stated no
> mechanism claim at the interview; strike this line and the sentence under it if
> it is not the claim being made._

We expect a product thinker to be able to read the shipped state from the brief
alone because the brief cannot fall behind a surface that has already shipped —
the gate refuses the shipped move and the release cut until the chapter names it
— and a shipped intent found later whose surface no chapter describes is what
would show the expectation wrong.

## Scope Conditions

None stated.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** the binary ships a verb, a sub-verb or an agent that no brief chapter
  names, **when** the deterministic layer runs, **then** it refuses and names the
  undocumented surface, with no oracle call and no wait on a reviewer.
- **Given** a change that delivered a surface, **when** the intent moves to
  shipped through `abcd spec close`, **then** the semantic layer reads what the
  change delivered against that surface's chapter and refuses the move on a
  confirmed false sentence, naming the sentence.
- **Given** the semantic reviewer cannot be reached or returns no usable verdict,
  **when** either enforcement point runs, **then** the gate refuses, rather than
  reporting the brief clean.
- **Given** a release cut, **when** the gate runs over every intent shipped since
  the last cut, **then** the cut is refused while any of their chapters lags,
  naming the intent and the lagging sentence.
- **Given** the gate finds the brief lagging, **when** the shipping change is
  prepared, **then** the brief edit is drafted and applied in that same change
  and flagged for the maintainer's review, so no change lands with the brief left
  lagging and unflagged.
- **Given** a public-doc sentence that lags the brief, **when** either
  enforcement point runs, **then** it is reported and the move or the cut
  proceeds — in this rung the public docs never refuse.
- **Given** a task has completed, **when** the per-task pass runs, **then** it
  reports where the brief may lag, with evidence pointing at the divergence, and
  blocks nothing.
- **Given** a brief edited after a release was cut, **when** the next cut's gate
  runs, **then** the edit is treated as the legitimate lead and nothing between
  the cuts reported the brief as wrong for being ahead.
- **Given** the gate is invoked with its autonomous flag inside an unattended
  run, **when** it finds a lagging chapter, **then** it drafts and applies the
  edit, lists every applied edit in the run's own output for review after, and
  waits for nobody; it still refuses on an undocumented surface, on a confirmed
  false sentence, and when the reviewer is unavailable.

## Open Questions

> _Every question below was put to the maintainer at the planning interview on
> 2026-09-01 and answered there; the answers are the rulings in the Decisions
> section above. The section is kept so a later reader sees what was asked._

- **Resolved — where the pass hooks.** Two points, one mechanism: `abcd spec
  close`, which moves the intent to shipped, and the release cut. Not a
  composition with the intent-fidelity reviewer's surface; a distinct gate at
  both moments.
- **Resolved — what "built reality" is as a concrete input.** Two layers: the
  command tree and the agent set for the deterministic layer, and what the change
  delivered for the semantic layer.
- **Resolved — the developer-extending public-doc view.** Out of scope. The
  public docs as a whole are a later rung; neither audience view ships here.
- **Resolved — how the per-task tier avoids noise.** By not being a gate. The
  per-task pass is kept as a report only; the refusal lives at the shipped move
  and the cut, where a change has a surface to be judged against.
- **Open, and not gating scope** — whether this pass becomes a framework-provided
  discipline (see [[itd-62-pluggable-safety-gate]]'s two-discipline-kinds
  question) or stays a reviewer surface like fidelity. Deferred: the answer
  changes where the pass is configured, not what this rung builds.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-7a2c62ab0228 -->
Fidelity review — receipt rcp-7a2c62ab0228 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:1ff0dc6b7d83c46b223640fffda8404d298f6607b181dd32f8a2caef0cab6cba
Input attestations: diff:commits 5992b3846 02ec15acb 662f9c81a b0510748f 7d50be159 9ddfe9d2b d155841e7 accf61856 cf3d3c9a1 cdd78b03b (PRs #? docFidelity + integ 24c), judged at main 7fb52a6b5@sha256:fa44e8580b3d8be5491d02703fc5ee9cd6a692fa46803d4ab5ead27bc906cd4a;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: Layer 1 derives every non-hidden, non-moved verb and sub-verb from the live command tree plus every agents/*.md, checks each against the chapter files for a code span of its path, refuses before the reviewer is constructed and names the surface in the reason; the spec-close test proves a matching PROMOTE does not excuse a missing chapter. Concern: the chapter set is every regular *.md under 04-surfaces/, so the index README.md counts as a chapter; at BASE `abcd rules`, `abcd spec`, `abcd spec close` and `sota-researcher` are named by no numbered chapter and pass the floor on the README alone, and a one-row index entry would satisfy the floor for any new surface.
  evidence: internal/core/docfidelity/docfidelity.go:209-229 — "if layerOne && !report { v.Refuse = true; return v }"
  evidence: internal/core/docfidelity/docfidelity.go:220 — "no brief chapter under 04-surfaces/ names the "+string(s.Kind)+" `"+s.Name+"`""
  evidence: internal/core/docfidelity/store.go:68-80 — "names, err := regularMarkdown(root, ChaptersDir) ... in.Chapters[name] = string(data)"
  evidence: internal/core/docfidelity/docfidelity_test.go:44 — "func TestLayerOneCoverageTable"
  evidence: internal/surface/cli/docfidelity_close_test.go:81-92 — "Even a matching PROMOTE review does not excuse a missing chapter."
  evidence: .abcd/development/brief/04-surfaces/README.md:241 — "`abcd rules` renders the rule set the prompt router draws on, read-only."
- ac-2 — MET_WITH_CONCERNS: `spec close` runs the gate over the intents the close would ship before anything moves, and a saved HOLD review refuses naming each false brief sentence with its evidence; the test asserts the spec stays open and the intent planned. Concern: the semantic layer is not invoked at the close — it is a receipt the host saved earlier with `docs fidelity record`, labelled by HEAD, and it reviews the chapters against the code as a whole, with the population only naming the findings; this shape rests on ruling DR3, which the code cites but the committed decision log does not carry.
  evidence: internal/surface/cli/cli.go:3463 — "if err := enforceDocFidelity(repoRoot, "abcd spec close", closeShips(repoRoot, args[0])); err != nil {"
  evidence: internal/core/docfidelity/docfidelity.go:253-271 — "case ReviewHold: ... reason := who + "the docs review confirmed a false sentence in " + f.Chapter + ": \"" + f.Sentence + "\" (" + f.Evidence + ")""
  evidence: internal/core/docfidelity/store.go:112-127 — "SavedReview is layer 2 as ruled: the review the delegated reviewer saved for Commit, found in Root."
  evidence: internal/surface/cli/docfidelity_close_test.go:68 — "func TestSpecCloseRefusesOnAConfirmedFalseSentence"
  evidence: internal/core/docfidelity/store.go:4-5 — "the one writer of that review (ruling DR3)"
- ac-3 — MET: Every way of not having a usable verdict is a refusing status: no receipt (none), a receipt for another commit (stale), an unreadable or hand-edited one (invalid), a non-PROMOTE non-HOLD verdict (inconclusive), a HOLD naming nothing, and a PROMOTE that names a false brief sentence; the core table, the close tests and the cut tests all assert the refusal, and Record refuses a PROMOTE-with-findings payload before it is saved.
  evidence: internal/core/docfidelity/docfidelity.go:248-283 — "case ReviewNone: ... case ReviewStale: ... case ReviewInconclusive: ... default: ... is refused"
  evidence: internal/core/docfidelity/store.go:116-117 — "Every way of not having a usable verdict is a status that refuses; none is a pass."
  evidence: internal/core/docfidelity/store.go:270-272 — "a PROMOTE names %d false brief sentence(s), so the brief is not current: a verdict with a confirmed brief sentence is HOLD"
  evidence: internal/core/docfidelity/docfidelity_test.go:117 — "func TestLayerTwoOutcomes"
  evidence: internal/surface/cli/docfidelity_close_test.go:56 — "func TestSpecCloseRefusesWithNoSavedDocsReview"
  evidence: internal/core/release/docfidelity_test.go:34 — "func TestEmitRefusesACutWithNoSavedDocsReview"
- ac-4 — MET_WITH_CONCERNS: The release emit collects every itd- record added since the base tag and runs the same Gate over them; a refusal carries the population and each reason, which name the intent and the lagging sentence; the cut test asserts it. Concern: the gate runs only when at least one intent shipped, so a cut carrying only issue resolutions runs neither layer — including layer 1, which the spec says is cheap enough to run at every enforcement point; a sub-verb added by an issue fix in such a cut meets no coverage floor at the cut.
  evidence: internal/core/release/emit.go:266-285 — "if len(shipped) > 0 { fidelity, armed, err := docfidelity.Gate(root, current.Commands, shipped, false)"
  evidence: internal/core/release/emit.go:278-281 — "Kind: RefusalDocFidelity, Reason: "the brief lags a surface shipped in this cut: " + strings.Join(fidelity.Reasons, "; "), Records: shipped,"
  evidence: internal/core/release/docfidelity_test.go:46 — "func TestEmitRefusesACutWhileAShippedIntentsChapterLags"
  evidence: internal/core/release/docfidelity_test.go:71 — "func TestEmitWithNoIntentShippedJudgesNoPopulation"
  evidence: internal/surface/cli/docfidelity.go:55 — "if len(population) == 0 || !docfidelity.Armed(repoRoot) { return nil }"
- ac-5 — MET_WITH_CONCERNS: A HOLD finding with a reviewer-drafted `replacement` becomes a proposed edit; the gate still refuses and names `--apply`; Apply replaces the sentence (exactly once, one bounded line, or nothing is written) and records a flag in .abcd/work/brief-review-flags.json, which travels in the change's diff; the judge then treats the sentence as applied only when the chapter changed AND a flag names it, so no path completes with the brief lagging and unflagged. Concern: the drafting is the reviewer's optional `replacement`, not the pass's own, and the apply is a separate `docs fidelity --apply` invocation rather than a step of the shipping verb; a HOLD with no drafted replacement is refused with nothing drafted.
  evidence: internal/core/docfidelity/docfidelity.go:266-269 — "if f.Replacement != "" { v.Proposed = append(v.Proposed, Edit{...}); reason += "; the reviewer drafted its correction — `abcd docs fidelity --apply` applies it and flags it for review" }"
  evidence: internal/core/docfidelity/apply.go:61-117 — "func Apply(root string, edits []Edit, commit string, at time.Time) ([]Flag, error)"
  evidence: internal/core/docfidelity/apply.go:28 — "const FlagsPath = ".abcd/work/brief-review-flags.json""
  evidence: internal/core/docfidelity/docfidelity.go:303-317 — "All three, so neither a silent hand edit nor a flag over an unedited chapter completes the change with the brief lagging and no flag recorded."
  evidence: internal/core/docfidelity/apply_test.go:51 — "func TestAnEditWithoutAFlagStillRefuses"
  evidence: internal/surface/cli/docfidelity_verb_test.go:55 — "func TestDocsFidelityApplyAppliesTheDraftAndTheCloseProceeds"
- ac-6 — MET: A finding with doc "public" is collected into public_findings and skipped by the HOLD loop, so it adds no reason and refuses nothing at either point; Record accepts the public doc kind and requires a HOLD to name a brief sentence, so a public-only review is a PROMOTE that is reported; the core test asserts the finding is present and the verdict allows.
  evidence: internal/core/docfidelity/docfidelity.go:232-236 — "if f.Doc == DocPublic { v.Public = append(v.Public, f) }"
  evidence: internal/core/docfidelity/docfidelity.go:255-258 — "if f.Doc == DocPublic { continue }"
  evidence: internal/core/docfidelity/store.go:267-268 — "a HOLD names no false brief sentence, so there is nothing to correct: a verdict with no confirmed brief sentence is PROMOTE"
  evidence: internal/core/docfidelity/docfidelity_test.go:160 — "func TestPublicDocSentenceIsReportedAndNeverRefuses"
- ac-7 — MET: `docs fidelity --report` runs both layers whatever layer 1 found, states every finding with its evidence, sets Refuse false and exits 0; it refuses to combine with --apply or --autonomous so it can write nothing.
  evidence: internal/core/docfidelity/docfidelity.go:223-229 — "layerOne := len(v.Reasons) > 0; if layerOne && !report {"
  evidence: internal/core/docfidelity/docfidelity.go:284 — "v.Refuse = !report && len(v.Reasons) > 0"
  evidence: internal/surface/cli/docfidelity.go:101-103 — "--report blocks nothing and writes nothing, so it takes neither --apply nor --autonomous"
  evidence: internal/core/docfidelity/docfidelity_test.go:176 — "func TestReportModeReportsTheSameFindingsAndNeverRefuses"
  evidence: internal/surface/cli/docfidelity_verb_test.go:28 — "func TestDocsFidelityReportModeStatesFindingsAndExitsZero"
- ac-8 — MET: Both enforcement points derive the command tree from the live binary (SurfaceSnapshot at the close, current.Commands at the cut) and the receipt is keyed on HEAD, so the brief is judged against the binary and never against the tag; the core test pins that a chapter ahead of the tag is not a finding.
  evidence: internal/core/release/emit.go:262-265 — "The brief is judged against the binary, never the tag, so a chapter edited ahead of the last cut is current, not drift."
  evidence: internal/surface/cli/docfidelity.go:58-62 — "snap, err := SurfaceSnapshot(repoRoot) ... docfidelity.Gate(repoRoot, snap.Commands, population, false)"
  evidence: internal/core/docfidelity/docfidelity_test.go:201 — "func TestTheLegitimateLeadIsNotDrift"
- ac-9 — MET_WITH_CONCERNS: `docs fidelity --autonomous` applies every drafted edit, lists each applied edit and its flag in the output, adds the reviewer's request (commit, population, chapters, record verb, verdict shape) to the JSON, and still exits 1 with no saved review, an undocumented surface or a confirmed false sentence; the test asserts the request and the refusal. Concern: the flag lives on the standalone verb only — `spec close` and `launch ship` have no autonomous form, and the launch page's cut routine names only `docs fidelity record`, never the autonomous verb — so an unattended cut waits for nobody only if its routine runs `docs fidelity --autonomous`, the reviewer and `record` itself before the cut.
  evidence: internal/surface/cli/docfidelity.go:123-145 — "if (apply || autonomous) && len(v.Proposed) > 0 && v.Review != nil { flags, err := docfidelity.Apply(...) ... if autonomous && (v.Review == nil || v.Review.Status != docfidelity.ReviewMatch) { ... req := docfidelity.NewRequest(head, in)"
  evidence: internal/surface/cli/docfidelity.go:149-151 — "if v.Refuse { return &exitError{Code: 1} }"
  evidence: internal/core/docfidelity/apply.go:119-129 — "Request is what an unattended routine hands the delegated reviewer, the way `launch ship` hands the changelog composer its request block"
  evidence: internal/surface/cli/docfidelity_verb_test.go:79 — "func TestDocsFidelityAutonomous"
  evidence: commands/launch.md:421-432 — "The cut needs a docs review for the commit it runs on. ... "${CLAUDE_PLUGIN_ROOT}/abcd" docs fidelity record --verdict-json verdict.json"

Gap audit:
- honoured:
  - Two layers, one gate: layer 1 refuses on its own with no reviewer constructed, layer 2 is host-delegated and judged by the release gate's own receipt reader
    evidence: internal/core/docfidelity/docfidelity.go:196-229 — "judge composes the verdict: layer 1 first, and only when it holds, layer 2."
    evidence: internal/core/docfidelity/store.go:128 — "got, err := lint.CheckGateReceipt(s.Root, ReceiptsDir, s.Commit, GateName)"
  - Two enforcement points share one entry point with different populations: the intents a close would ship, every intent shipped since the tag
    evidence: internal/core/docfidelity/store.go:185-189 — "Gate is the one entry point both enforcement points and the per-task report call, with their own population"
    evidence: internal/surface/cli/docfidelity.go:27-49 — "func closeShips(repoRoot, specID string) []string"
  - Fail closed: no reviewer, no comparison, no verdict all refuse, and the three fail-open shapes found after the first landing are closed
    evidence: internal/core/docfidelity/docfidelity.go:239-247 — "a receipt edited to carry one after it was saved is still not a pass"
    evidence: internal/core/docfidelity/store_test.go:201 — "func TestRecordRefusesAFailOpenPayload"
  - The judge writes nothing; the writer is a distinct Apply the judge never reaches
    evidence: internal/core/docfidelity/apply.go:3-8 — "A judge that writes is a judge whose output depends on who ran it, so nothing here is reached by Judge, and nothing in Judge writes."
    evidence: internal/core/docfidelity/store_test.go:184 — "func TestGateWritesNothing"
  - Brief only: a public-doc sentence is reported and never refuses
    evidence: internal/core/docfidelity/docfidelity.go:255-258 — "if f.Doc == DocPublic { continue }"
  - The legitimate lead is not drift: the brief is judged against the binary, never the tag
    evidence: internal/core/docfidelity/docfidelity_test.go:201 — "func TestTheLegitimateLeadIsNotDrift"
  - A remainder close ships nothing and is not gated
    evidence: internal/surface/cli/docfidelity_close_test.go:106 — "func TestSpecCloseWithARemainderIsNotGated"
  - No file exempts a surface from layer 1: the backlog file the first landing read was removed
    evidence: internal/core/docfidelity/store_test.go:241 — "func TestNoBacklogFileAdmitsAnUndocumentedSurface"
  - Wired on both front doors and documented in the brief's own docs chapter
    evidence: commands/docs.md:75-120 — "## `fidelity` — the brief describes every surface that ships"
    evidence: .abcd/development/brief/04-surfaces/10-docs.md:66-100 — "**The doc-fidelity gate** (itd-60) judges whether the brief describes every surface that ships."
- diverged:
  - The semantic layer was promised as a review the close and the cut run over what the change delivered; delivered is a receipt the host saves beforehand for HEAD, found automatically, reviewing the chapters against the code as a whole — the shape ruling DR3 chose, which the committed decision log does not carry
    evidence: internal/core/docfidelity/store.go:3-10 — "the delegated docs review saves a verdict receipt labelled with the commit it reviewed; `spec close` and `launch ship` find it automatically"
    evidence: .abcd/work/DECISIONS.md:2614 — "The person answered eighteen rulings of autonomous run A on 2026-09-30 (CF1, CF2, CG1, CI1, CI2, CJ1, CK1, CL1, CM1, DR1, DR2, DR4, DR5, DR6, DQ1a, DQ1b, DQ2, DQ3)"
  - Layer 1 treats every regular *.md under 04-surfaces/ as a chapter, the index README.md included; four surfaces at BASE are covered by the index alone
    evidence: internal/core/docfidelity/store.go:70 — "names, err := regularMarkdown(root, ChaptersDir)"
    evidence: .abcd/development/brief/04-surfaces/README.md:262 — "`abcd spec` addresses the spec store under `.abcd/development/specs/`"
  - The deterministic floor was promised at every enforcement point; both points skip the whole gate when the population is empty, so a cut or close that ships no intent runs no coverage check
    evidence: internal/core/release/emit.go:272 — "if len(shipped) > 0 {"
    evidence: internal/surface/cli/docfidelity.go:55 — "if len(population) == 0 || !docfidelity.Armed(repoRoot) {"
  - The pass was to draft the brief edit itself; delivered, the reviewer drafts an optional replacement and a separate --apply invocation applies it
    evidence: internal/core/docfidelity/docfidelity.go:50-51 — "Replacement is the reviewer's drafted sentence, "" when none was drafted."
  - The autonomous flag lives on the standalone verb; the launch page's cut routine does not name it
    evidence: internal/surface/cli/docfidelity.go:157-158 — "cmd.Flags().BoolVar(&autonomous, "autonomous", false,"
    evidence: commands/launch.md:421-427 — "refuses (`doc-fidelity`, "run the docs review first") until a review is saved for HEAD"
- missing:
  - A chapter of their own for `abcd spec` (and `abcd spec close`), `abcd rules` and `sota-researcher`: no numbered chapter under 04-surfaces/ names them, so their flags appear in no generated appendix, and the gate did not refuse because the index names them
    evidence: .abcd/development/brief/04-surfaces/README.md:241 — "`abcd rules` renders the rule set the prompt router draws on, read-only."
    evidence: internal/core/docfidelity/docfidelity.go:211-216 — "for _, name := range chapters { if names(in.Chapters[name], s) { row.Chapter = name; break } }"
<!-- abcd-review-end receipt=rcp-7a2c62ab0228 -->

- 2026-10-02: the press release's sentence and the two Decisions sites (the opening line and the draft-and-apply bullet) name the product thinker, the role each sentence means, in place of the retired role word, under the product thinker's ruling R5 of that day, recorded as iss-2610021446271464; every other word is unchanged, and the retired word's remaining occurrences elsewhere in this record were outside the ruling.

## References

- Sequenced by
  [Phase 8 — the brief is the shipped state](../../roadmap/phases/phase-8-brief-currency.md),
  whose expectation this intent's mechanism claim is derived from, and whose
  milestone names this pass as the hard gate at the shipped move.
- Rests on adr-5: the brief is the current state of the project. This intent is
  what makes that decision enforceable.
- Originating assessment: `~/Desktop/abcd-assessment.html` (2026-06-26) — the
  README-over-promises / brief-honest finding that motivates the forward
  doc-fidelity loop. Its public-docs half is the later rung.
- Builds on itd-80, the intent-fidelity reviewer (delivery-against-intention):
  this pass is the documentation-against-delivery analogue, it consumes that
  reviewer's notion of what a change delivered, and it reuses its fail-closed,
  deterministic-shell, never-silent posture.
- Builds on itd-73, derived versioning, whose structural surface diff already
  runs at the release cut — the second of this gate's two enforcement points.
- Paired with: [[itd-61-brief-change-derivation]] (the reverse direction) and
  [[itd-62-pluggable-safety-gate]] (whose brief change this pass would govern);
  the other Phase 8 rung is
  [[itd-147-the-brief-s-surface-chapters-are-a-generated-reflection-of-t]].
- Governing principle: single source of truth — the brief is canonical
  (`.abcd/development/brief/` and the engineering conventions in `AGENTS.md`).

## Grounds

- pursued: the brief is the shipped state, so a product thinker reads the shipped state from the brief alone; a shipped intent whose surface the brief does not describe, found after the gate exists, shows the gate was wrong
