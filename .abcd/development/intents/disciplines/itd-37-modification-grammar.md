---
id: itd-37
slug: modification-grammar
kind: discipline
kind_notes: "Cross-cutting modification-grammar gate; applied at every spec plan-review and ship time via concreteness lint (MG001-MG003) + intent-auditor Role 1 boilerplate detection (MG004). Every spec inherits this rule. Closes Naur's Modification axis — the genuinely new gap in abcd's theory transmission (Mapping + Justification already partially captured by press release + audit notes)."
suggested_kind: null
spec_id: null
reclassification_history: []
builds_on: [itd-1, itd-36]
severity: major
---

# Every Spec Externalises How It Should And Shouldn't Be Modified

## Rule

Every spec carries a `## Modification Grammar` section with three sub-headings, each of which MUST be non-trivial and non-boilerplate:

1. **`### Extends cleanly`** — at least one concrete example with a constraint. *"What kind of change can someone make to this spec's deliverables without breaking the design?"* Concrete = at least one fenced code block OR file-path reference OR line-number reference; the constraint names a specific extension point.
2. **`### Breaks the design`** — at least one named failure mode. *"If someone does X, the design breaks because Y."* Concrete failure mode, not "general code-quality concerns" or "anti-patterns to avoid."
3. **`### Why`** — the structural rule from which both sub-headings follow. The rule, not a list of cases — a list of cases without an underlying rule fails the discipline. The rule must be specific enough that stripping the spec name still identifies the rule's spec.

A fourth axis — **`### Ripple`** — captures the per-spec-external concerns absorbed from idea-3 (systems thinking) at adversarial review (chat `idea-3-itd-38-adversaria-31A06A`, MAJOR_RETHINK outcome): vocabulary delta (HARD-enforced via the [vocabulary-registration requirement in `02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md)), surface delta (new commands / sub-verbs / agents / disciplines), coupling delta (what this spec newly depends on; what newly depends on it). The `Ripple` axis stays under itd-37 (same retrieval key as modification grammar — domain) rather than spawning a separate discipline.

At spec completion, `principle-distiller` (the role-extended curator from [itd-36](../shipped/itd-36-memory-unification.md)) extracts the `## Modification Grammar` section into two memory pages per [`05-internals/07-memory.md`](../../brief/05-internals/07-memory.md):

- **`spec_modification_grammar_<spec_id>.md`** — append-only, per-spec. Source class `spec_modification_grammar`. Lifecycle: append-only per [`05-internals/04-universal-patterns.md § 8`](../../brief/05-internals/04-universal-patterns.md#8-artefact-lifecycle-taxonomy).
- **`modification_grammar_<domain>.md`** — compounding-curated, per-domain. `principle-distiller` runs a curator-pass that synthesises across the per-spec pages. Lifecycle: compounding-curated per the same taxonomy.

Two page classes, two lifecycles, two retrieval surfaces. The per-spec page preserves provenance; the per-domain page is the cross-cutting query surface that aggregates across specs in the same domain.

## Why

abcd's three-layer mental model captures *what* (brief), *why* (intent), *how to build* (spec). It does not, until itd-37, capture *how to extend without breaking the structure*. Per Naur ("Programming as Theory Building", 1985, [gwern.net/doc/cs/algorithm/1985-naur.pdf](https://gwern.net/doc/cs/algorithm/1985-naur.pdf)), there are three areas of tacit knowledge code cannot capture: Mapping (what the program represents), Justification (why each load-bearing decision), and **Modification (the structural rules that govern modification)**. abcd already covers Mapping (press release + acceptance criteria) and Justification (audit notes + open questions + brief decisions + reclassification history) partially. **Modification is the genuinely new gap.**

The discipline is named for what's actually new — **modification grammar**, not "theory" writ large. The renaming was forced by adversarial RP review (chat `adversarial-naur-theory--0A4174`): "Theory" overpromised (the section captures a *subset* of theory, not theory writ large; an engineer seeing `## Theory` guesses "academic justification" or "literature review" — both wrong); "modification grammar" is honest, specific, and forces concrete content.

**Why two-layer enforcement.** Boilerplate-rot is the discipline's principal failure mode. A `## Modification Grammar` section that says *"extending requires care; modifying requires understanding the design"* is non-empty, parseable, content-free — and **worse than no section at all** because it occupies cross-cutting query surface with noise. A regex-level "non-empty" lint catches the laziest failure but not plausible-sounding boilerplate. Two layers:

1. **Concreteness lint (mechanical)** rejects sections without a concrete reference (code block / file path / line ref). Lint codes `MG001`-`MG003`.
2. **Semantic boilerplate detection (reviewer judgement)** runs `intent-auditor` Role 1 (the discipline role per itd-1) at spec plan-review and ship time. Prompt-encoded test: ***"Strip the spec name. Could this `## Modification Grammar` text describe a different spec? If yes, reject."*** Lint code `MG004`.

The semantic enforcement is genuine LLM-judgement work. The discipline owns the requirement explicitly — regex cannot catch boilerplate.

**`MG004` enforcement surface (made concrete by spc-12).** The timing is unchanged — `MG004` runs at plan-review and ship time — but the named surface is the abcd-owned CI / pre-commit path: the native disciplines lint/CI wrapper invoked from `.github/workflows/ci.yml`, **not** a plan-review-step integration. The Role 1 `MG004` pass emits a `PASS` / `FAIL` boilerplate verdict; the verdict lands in a per-run batch receipt at `.abcd/logbook/audit/spec-mg-<ts>/report.{json,md}` (native specs have no `## Audit Notes` section, so the verdict cannot land in-file). spc-12 ships the `MG004` judgement pass, its receipt writer, and the receipt schema; the CI wiring is added by spc-12 itself via the native disciplines lint.

**Why the cost is justified.** itd-37 is the first *expensive* discipline in abcd's stack (~15-30 min careful thought per spec, vs ~5 min for itd-1 / itd-5 / itd-36's mechanical gates). With itd-1 + itd-5 + itd-36 + itd-37 all live, every spec carries 4 discipline gates costing ~30-45 min total — real, but justified. The failure modes the disciplines prevent (specs without acceptance bars, agents without quality gates, specs without modification grammar, specs without provenance) compound exponentially as the corpus grows. Disciplines are a fixed per-spec tax; the failures are exponential. Trade favourably.

**Why all specs, no "trivial" carve-out.** "Trivial" cannot be cleanly specified without inviting loophole-driven exemptions. A `## Modification Grammar` section that says *"this spec adds a config flag with no extension surface; modifications are limited to renaming the flag; the rule is: never bind config-flag names to public API"* is **valuable** — documenting absence-of-extension-points IS modification grammar. Required for all; trivial specs produce short Modification Grammar sections, not absent ones.

**Why memory routing as secondary index, not primary store.** The dominant access pattern is "I'm modifying spec spc-N, give me spc-N's modification grammar" — that's a lookup the spec file already serves trivially. Memory routing earns its keep on the long-tail: *"how has our modification grammar of dispatch evolved across spc-1, spc-3, spc-7?"* Cheap to add (`principle-distiller` already exists post-itd-36); worth it; not load-bearing for the common case. In-spec capture is the primary store. This is also why itd-36 is a `builds_on` edge, not a blocker: the capture + enforcement half (`MG001`-`MG004`, the Phase 0 discipline registration) ships independently of itd-36, and only the extraction-to-memory trigger waits on its substrate — the partial-ship fallback below is the designed behaviour, not a contingency.

## What's In Scope

- **`## Modification Grammar` section template** carried by the native spec template in the Go binary (`internal/core/...`) with three sub-headings (`Extends cleanly` / `Breaks the design` / `Why`) plus the `Ripple` axis (vocabulary / surface / coupling deltas), one worked example per axis. Section header is fixed (parser depends on it).
- **Concreteness lint** in the native intent lint with three codes:
  - `MG001` — section missing.
  - `MG002` — sub-heading missing or empty.
  - `MG003` — sub-heading present but contains no concrete reference (code block / file path / line ref).
- **Semantic boilerplate detection** via `intent-auditor` Role 1's discipline-checking pass. New prompt-encoded test ("could this describe a different spec?"). Lint code `MG004` — Role 1 boilerplate verdict.
  - **Specific rejection criteria** the prompt encodes: (a) `Extends cleanly` rejected if it doesn't name a concrete extension point with a constraint; (b) `Breaks the design` rejected if it doesn't name a specific failure mode; (c) `Why` rejected if it's a list of cases without an underlying rule, OR if the rule could equally describe a different spec.
- **`principle-distiller` extraction trigger** — at spec completion, the curator (per itd-36's role extension):
  - Writes append-only `spec_modification_grammar_<spec_id>.md` to `.abcd/memory/` with `source.class: spec_modification_grammar`.
  - Updates curator-merged `modification_grammar_<domain>.md` with `source.class: modification_grammar` (compounding-curated).
- **Recovery-humility paragraphs** on `/abcd:disembark` and `/abcd:embark` surfaces (≤4 sentences each). Disembark: lifeboat is the floor of recoverable theory, not theory itself. Embark: hunt the originating session before trusting the lifeboat blindly. Already landed in [`04-surfaces/02-disembark.md`](../../brief/04-surfaces/02-disembark.md) and [`04-surfaces/03-embark.md`](../../brief/04-surfaces/03-embark.md) per Layer A brief edits.
- **Mental-model "Naurian gap" sub-section** — added at [`01-product/03-mental-model.md`](../../brief/01-product/03-mental-model.md). Names the three Naur axes; identifies Modification as the genuinely new gap; cites Naur 1985.
- **Vocabulary-registration requirement** (HARD) — every term introduced in `### Ripple > Vocabulary delta` MUST appear in the same spec in the registry that fits it, per the routing rule in [`02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md): the canonical glossary for cross-cutting prose vocabulary, that file's reserved-vocabulary register for controlled enums. The intent lint is designed to block at plan-review with code `VR001` (reserved; no `SpecLinter` implementation exists yet, so registration is enforced by review today).
- **Karpathy & Naur citations** in `research/related-work.md` (Karpathy as pattern source for itd-36; Naur as philosophical citation for itd-37). Both in [`research/related-work.md`](../../research/related-work.md).
- **Inheritance into every spec** — every native spec plan-reviewed under abcd inherits this discipline's gates: spec must include `## Modification Grammar`; concreteness lint runs at plan-review; Role 1 boilerplate detection runs at plan-review and ship time.

## What's Out of Scope

- **Naur's "theory layer" framing in mental-model** — the mental-model file is for axes (brief / intents / specs), not components. Adding "the brief is itself a theory layer" conflates orientation with implementation, and the citation is a philosophical anchor that belongs in `research/related-work.md`, not in the project's most-stable orientation file. This was held against in idea-2 R3 review.
- **Auto-classification of "trivial" specs that can skip the discipline** — explicitly rejected. Trivial-self-exemption risk dominates; trivial specs produce short Modification Grammar sections (e.g., "this spec adds a config flag; modifications limited to renaming"), not absent ones.
- **Full ADR-style decision documentation** — `## Modification Grammar` is the 1-page "what the theory of this spec is" so the next agent doesn't fly blind. Full ADRs (multi-page architectural decision records with stakeholders, alternatives evaluated, consequences) live at [`.abcd/development/decisions/adrs/`](../../decisions/adrs/) and are not affected by this discipline.
- **Retroactive backfill of specs that ship before itd-37 lands** — itd-36 + itd-37 ship together (per the contingent-pairing in idea-2 R5 review). If itd-36 + itd-37 ship first, all subsequent specs inherit the discipline. If itd-37 slips relative to itd-36, the partial-ship fallback (capture in spec + reviewer enforcement; extraction-to-memory deferred) covers the gap.
- **Cross-intent modification grammar dependencies** — each spec's modification grammar is independent. No "this spec's modification grammar inherits from itd-N" beyond the discipline's own application.
- **Idea-3 standalone discipline (itd-38 system-impact)** — explicitly absorbed as the `Ripple` axis (idea-3 R1 reviewer ruling: same retrieval key, same audience, same lifecycle as modification grammar; split was rhetorical not architectural). itd-38 ID released, not reserved.

## Acceptance Criteria

> _BDD format, per [itd-1 acceptance gates](itd-1-acceptance-gates.md). These gates are checked by `intent-auditor` Role 1 against every spec plan-reviewed under abcd._

- **Given** a spec without a `## Modification Grammar` section, **when** plan-review runs, **then** the intent lint emits `MG001` and blocks promotion.
- **Given** a spec with `## Modification Grammar` but missing one of the three required sub-headings (`Extends cleanly` / `Breaks the design` / `Why`), **when** plan-review runs, **then** the lint emits `MG002` naming the missing sub-heading.
- **Given** a spec where any of the three sub-headings is present but contains no concrete reference (no code block, no file path, no line ref), **when** plan-review runs, **then** the lint emits `MG003` and points at the offending sub-heading.
- **Given** a spec where `## Modification Grammar` content could equally describe a different spec (boilerplate failure), **when** `intent-auditor` Role 1 runs the discipline check, **then** the reviewer emits `MG004` with the rejection reason ("strip-the-name test fails: this prose describes [generic concern] not [this spec's specifics]").
- **Given** a spec with `### Ripple > Vocabulary delta` introducing a new term not registered in either registry named by `02-constraints/04-naming.md`, **when** plan-review runs, **then** the intent lint is designed to emit `VR001` and block promotion until the term is registered (reserved; not yet implemented — today this is a review-time check, not a lint gate).
- **Given** a spec transitions to shipped, **when** `principle-distiller` runs the extraction pass, **then** an append-only memory page `spec_modification_grammar_<spec_id>.md` is written to `.abcd/memory/` with `source.class: spec_modification_grammar` AND a curator-merged page `modification_grammar_<domain>.md` is updated with `source.class: modification_grammar`.
- **Given** the cost-discipline boundary (itd-37 is first expensive discipline; ~15-30 min capture per spec), **when** any spec proposes adding "modification grammar exemption for trivial specs", **then** the proposal is rejected — trivial-self-exemption invites loophole-driven bypass; trivial specs produce short Modification Grammar sections, not absent ones.
- **Given** the partial-ship fallback (itd-36 ships only `ingest`/bare without `ask`/`lint`, OR `principle-distiller` extraction trigger slips), **when** itd-37 lands, **then** capture (`## Modification Grammar` in a spec) and reviewer-Role-1 enforcement (`MG001`-`MG004`) survive independently; memory extraction (`spec_modification_grammar_*` and `modification_grammar_*` pages) defers gracefully.
- **Given** the brief's verification matrix at [`06-delivery/02-verification-matrix.md`](../../brief/06-delivery/02-verification-matrix.md), **when** itd-37 ships, **then** the matrix carries entries for: modification-grammar capture, boilerplate detection, extraction at spec completion, and vocabulary-registration enforcement.
- **Given** Naur's 1985 paper as the philosophical citation, **when** a contributor reads `01-product/03-mental-model.md § The Naurian gap`, **then** they find: the three axes (Mapping / Justification / Modification) named explicitly, the gap statement (Mapping + Justification already partially captured; Modification is the genuinely new piece), the Karpathy/Naur citation pointing at `research/related-work.md`, and the recovery-humility framing for disembark/embark.

## Open Questions

- **Discipline-set audit trigger** — with itd-37 shipping, there are 3 disciplines (itd-1 + itd-5 + itd-37); under the user's deferred meta-discipline trigger threshold of ≥5. itd-36 is `kind: standalone` (per the user's classification call on 2026-05-08), not counted as a discipline. Trigger doesn't fire yet; meta-discipline question stays dormant. Surfaces as a fresh idea if a new discipline lands OR contradiction is observed.
- **Lint code numbering collisions** — `MG001`-`MG004` (this discipline), `MQ001`-`MQ002` + `MS001`-`MS002` + `ML001` (itd-36), `VR001` (vocabulary-registration), `SD001` (bare-command-as-render discipline) all reserved post-2026-05-08. Verify against [`05-internals/06-lint.md`](../../brief/05-internals/06-lint.md) reservation table at promotion time.
- **principle-distiller overload** — three roles (memory curator + per-spec extractor + per-domain curator) on one agent. Soft-cap risk per the agent-count discipline (15 agents, role extension preferred over agent splitting per itd-31 precedent). If coherence breaks, fork the curator role into a sibling agent. Logged in itd-37's risk register; not a current blocker.
- **Recovery-humility wording length** — disembark/embark paragraphs ≤4 sentences each (already landed in Layer A brief edits with closing imperative). Verify they don't grow into a wall of caveats over time; Role 2 cross-document audit should catch drift.

## Audit Notes

_Empty. Populated by `intent-auditor` Role 1 (single-document fidelity per itd-1) when this discipline is first audited. Like itd-1, this discipline is audited continuously via the rule-applies-to-every-spec semantics rather than via a planned→shipped transition. The reviewer's findings here record any spec that violated the discipline (e.g., shipped without `## Modification Grammar`, or with boilerplate `MG004` not caught at plan-review)._

## References

- Full assessment in a local working note (unmigrated), with 3-round review trail (chat `adversarial-naur-theory--0A4174`); rename from "Theory transparency" to "Modification Grammar discipline" preserved in chat record.
- Sibling assessment in a local working note (unmigrated); the `Ripple` axis on this discipline absorbs idea-3's per-spec-external concerns (chat `idea-3-itd-38-adversaria-31A06A`, MAJOR_RETHINK collapse).
- [Naur 1985][naur-1985] — "Programming as Theory Building", *Microprocessing and Microprogramming* 15(5):253-261; primary philosophical citation.
- [`research/related-work.md § Naur 1985`](../../research/related-work.md#naur-1985--programming-as-theory-building) — full prior-art comparison.
- [`itd-1-acceptance-gates.md`](itd-1-acceptance-gates.md) — companion discipline; this discipline's acceptance criteria conform to its Given-When-Then shape.
- [`itd-5-prompt-quality-additions.md`](itd-5-prompt-quality-additions.md) — companion discipline; disciplines stack at three (itd-1 + itd-5 + itd-37).
- [`../shipped/itd-36-memory-unification.md`](../shipped/itd-36-memory-unification.md) — companion intent (standalone); ships alongside itd-37; provides the substrate for `spec_modification_grammar` and `modification_grammar` page classes.
- [`05-internals/07-memory.md`](../../brief/05-internals/07-memory.md) — substrate spec for the curator agent's two-page-class extraction.
- [`02-constraints/04-naming.md`](../../brief/02-constraints/04-naming.md) — vocabulary-registration requirement (HARD) and bare-command-as-render discipline; both companion rules to this discipline's `Ripple` axis.
- [`01-product/03-mental-model.md § The Naurian gap`](../../brief/01-product/03-mental-model.md) — the framing this discipline closes.
- [`04-surfaces/02-disembark.md`](../../brief/04-surfaces/02-disembark.md) and [`04-surfaces/03-embark.md`](../../brief/04-surfaces/03-embark.md) — recovery-humility paragraphs (Layer A brief edits).

[naur-1985]: https://gwern.net/doc/cs/algorithm/1985-naur.pdf "Naur (1985) — Programming as Theory Building, Microprocessing and Microprogramming 15(5):253-261"
