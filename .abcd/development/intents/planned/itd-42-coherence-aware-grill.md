---
id: itd-42
slug: coherence-aware-grill
spec_id: null
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
---

# Grill Reads an Intent Against the Brief and Its Siblings, Not Just the Glossary

## Press Release

> **abcd's grill stops checking an intent in isolation: a full grill now reads it against the brief's invariants, its scope boundary, and every other intent — and asks the coherence questions a solo interrogation cannot.** Capturing an idea stays one line and zero friction. But when a product thinker promotes a draft, the grill loads more than the terminology glossary: it loads the brief's invariants and scope sections, and a one-line index of every other intent — drafted, planned, and shipped. Now it can ask the question that actually catches wrong code: "Invariant 3 says config is never written outside `~/.abcd/` — your intent implies it is; which gives?" and "itd-19 already covers stage-aware behaviour — how is this different?" It still grills vague terms and hidden assumptions; it now also grills *incoherence* — the intent that is locally clear and globally wrong. Grilling against the corpus is grounded the way the phase negotiator is grounded: a coherence concern that cannot be tied to a named invariant, scope clause, or sibling intent is asked as a Socratic question, never asserted as a conflict.
>
> "I'd grill an intent and it would come out crisp — clear terms, testable acceptance — and still be quietly redundant with something I'd specced two months earlier," said Iris, product lead. "abcd asked me, straight out: 'itd-31 already does cross-document fidelity — what does this add?' It didn't decide for me. It made me say the difference out loud, or admit there wasn't one. I killed one intent and sharpened another, before either reached a spec."

## Why This Matters

The intent layer is abcd's highest-leverage moment for product clarity ([itd-27](itd-27-grill-skill-and-glossary.md) built `/abcd:intent grill` on exactly that premise). But the grill itd-27 shipped has a blind spot its `--with-docs` flag name actively hides: "glossary-aware" mode loads **only** the terminology database. It checks that an intent *speaks* the brief's words. It never reads the brief, never reads another intent, never reads a shipped spec. It enforces vocabulary; it does not enforce coherence. This intent also corrects the misnomer: `--with-docs` becomes `--glossary` (the terminology tier it always was), the new coherence tier is `--coherence`, and `--full` runs both.

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

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline._

- **Given** an intent in `drafts/`, **when** the product thinker runs a light grill on it, **then** the grill loads at most the glossary tier and does not load brief or sibling context — capture-stage grilling stays cheap.
- **Given** an intent being promoted out of `drafts/`, **when** the full grill runs, **then** it loads the glossary tier, the named brief invariant/scope/surface slices, the `principles/` set, and the one-line sibling-intent index.
- **Given** an intent whose body implies behaviour that a `02-constraints/03-invariants.md` invariant forbids, **when** the full grill runs, **then** it surfaces the conflict and cites the specific invariant; a conflict surfaced without such a citation is a defect.
- **Given** an intent that overlaps a sibling intent, **when** the full grill runs, **then** it names the sibling intent ID and asks the product thinker to state the difference as a Socratic question — sibling overlap is never asserted as fact, because a draft sibling is a mutable anchor.
- **Given** the product thinker answers a surfaced overlap question, **when** the session ends, **then** the answer — the stated difference, or a kill/merge decision — is captured in the `grill-report` against that question, so the claimed distinction is on record.
- **Given** a coherence concern the grill cannot anchor to a named invariant or scope clause, **when** it surfaces that concern, **then** it is phrased as a Socratic question tagged with a named move — never as an asserted conflict.
- **Given** a full grill has run, **when** the `grill-report.json` is written, **then** coherence questions and any grounded conflicts appear in it alongside the existing question stream, each grounded conflict carrying its invariant or scope-clause anchor reference.
- **Given** the grill has surfaced a coherence conflict, **when** the session ends, **then** the intent, the brief, and the sibling intents are unchanged on disk — the grill's coherence output is advisory.

## Open Questions

> _Tier-selection surface and the Tier 3 index mechanism were resolved during the grill (2026-05-16): tier set is lifecycle-derived with `--light`/`--full` override; the sibling index is built fresh each grill, not a maintained file. Brief-slice degradation is resolved in scope (skip-and-warn). The questions below remain genuine plan-time decisions._

- Does coherence grilling share a context-loading module with itd-41's phase negotiator and itd-31's cross-document fidelity reviewer, or keep its own loader until the three demonstrably converge?
- Should a surfaced sibling-overlap question, once the thinker answers it, be allowed to *recommend* reclassification (bundle-member) or supersession, or strictly surface-and-record? itd-27's grill already touches reclassification-adjacent territory.
- Token budget — at 40+ intents the one-line index is small, but the brief slices plus glossary plus intent body must still fit. Is there a point where Tier 2/3 must itself become selective (the itd-39 boundary)?

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Extends: [itd-27](itd-27-grill-skill-and-glossary.md) (grill skill & glossary) — adds a coherence tier to the grill itd-27 built; the glossary tier's behaviour is unchanged. **Also renames itd-27's `--with-docs` flag to `--glossary` and adds `--coherence` / `--full`** — itd-27's surface table and the grill `SKILL.md` flag list must be updated when this intent is planned.
- Shares the grounded-adversary pattern with: [itd-41](../drafts/itd-41-phase-negotiator.md) (phase negotiator) — Socratic where it questions, grounded where it asserts.
- Defers to: [itd-39](../drafts/itd-39-scope-aware-memory-retrieval.md) (scope-aware memory retrieval) — full-body cross-intent comparison at scale is itd-39's problem, not this intent's.
- Coordinates with: [itd-48](itd-48-intent-fidelity-reviewer-roles-2-3.md) (cross-document fidelity reviewer — supersedes [itd-31](../superseded/itd-31-cross-document-fidelity-reviewer.md)) — different register: itd-48's Role 2 reviews delivered documents for drift; this grills an intent for coherence before it is planned.
