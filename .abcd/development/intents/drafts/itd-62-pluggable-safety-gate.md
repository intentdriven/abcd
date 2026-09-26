---
id: itd-62
slug: pluggable-safety-gate
spec_id: null
kind: null
suggested_kind: standalone
reclassification_history: []
related_adrs: []
prd_path: null
grill_session_id: 62d0f1de-0003-4a62-9c0d-000000000062
glossary_terms_used:
- core/brief
- core/intent
- core/oracle
grilled_intent_hash: 07b637b032a89ea870fbf6d78d9f6dc88d0ec1a40e124811d6ac4e3d62c7ae2d
prd_grandfathered: false
builds_on: [itd-60, itd-61]
severity: major
---

# abcd Routes The Invisible Risks An Amateur Cannot See To A Real, Fail-Closed Gate That Wraps A Trusted Scanner

## Press Release

> **abcd gains a pluggable validation-gate framework: a new kind of discipline that wraps an unmodified external scanner, fails closed, and emits the same four-verdict result as the fidelity reviewer — shipped with security/dependency/secret scanning as its first instance, validating the *downstream app the product thinker builds*, not abcd's own pipeline.** The amateur-coder thesis says the risks a non-expert cannot see — most acutely security — must be routed to a gate whose own reliability is validated. abcd has built the intent→verdict spine but left this safety pillar as an extension point. This intent fills it without violating abcd's core rule: it does not fork or reimplement a scanner, it *configures and wraps* a trusted one. The wrapping machinery is generic; a11y, privacy, and architecture become later instances of the same pluggable pattern. It runs as a standalone CLI across CI, pre-commit, and runtime hooks — no Claude Code dependency — so it works on a local machine.

> "I can judge whether the app does what I asked. I cannot judge whether it just shipped an XSS hole or a leaking dependency — that's exactly the part I'm blind to," said Iris, a product thinker building a real app on abcd. "I don't want abcd to *describe* a safety gate. I want it to actually run a scanner on what I built and refuse to call it safe when it can't tell. And I want that to keep working when I'm not in Claude Code."

## Why This Matters

The amateur-coder thesis is explicit: keep the human's sense of "good" as the limiting constraint, and route what lies beyond it — security above all — to automated gates whose reliability is itself validated. An external assessment (2026-06-26) found this is abcd's single largest gap against its own thesis: the process-protecting gates (gitleaks, PII, push-block) are real and fail-closed, but they protect *abcd's own dev process*. The "invisible risks" of the **downstream app the amateur builds** — security, dependency, secret exposure, later a11y/privacy/architecture — are documented as a pattern, not delivered as a gate. The empirical case is stark: a large fraction of AI-generated code ships at least one OWASP vulnerability. A framework targeting amateurs that does not route this to a gate is, by its own logic, incomplete.

Two abcd governing rules shape *how* this gets built. First, **abcd never touches its dependencies** — it wraps and configures unmodified trusted tools, it does not fork, patch, or reimplement them. A safety gate must wrap an existing scanner, not become one. Second, **abcd must eventually run locally, independent of Claude Code.** The gate is therefore a standalone CLI exposed across the three tiers abcd already uses (CI, pre-commit, runtime hooks), reusing the fidelity reviewer's fail-closed, four-verdict posture so it composes with the machinery already trusted rather than inventing a parallel one. Building it as a *pluggable framework* (security first) rather than a one-off scanner means the remaining invisible risks plug into the same validated mechanism.

## What's In Scope

- A **pluggable validation-gate framework**: a new *kind* of discipline — framework-provided "validation disciplines" that wrap an external tool and fail closed — distinct from the existing app-authored disciplines. (Introducing this distinction is a brief change; per [[itd-60-doc-fidelity-anti-drift]] / [[itd-61-brief-change-derivation]] that brief change is itself governed by the doc-fidelity loop.)
- A **first instance: security / dependency / secret scanning**, wrapping an unmodified, trusted external scanner (never forked, never reimplemented), targeting the **downstream app the product thinker builds**.
- **Standalone CLI** as the primary entrypoint, invocable from CI, pre-commit, and runtime hooks — the same three-tier model abcd uses elsewhere — with **no Claude Code dependency**, so the gate runs on a local machine.
- **Fail-closed, four-verdict wiring:** the gate emits MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE per check, refuses (never passes silently) when the scanner is unavailable or a trustworthy scan cannot complete, and feeds the existing receipt/roll-up shape — mirroring the native ship gate and the fidelity reviewer.
- Per-project configuration of which validation disciplines apply, consistent with abcd's "downstream apps configure" model.

## What's Out of Scope

- **a11y / privacy / architecture gates themselves.** They are explicitly the *later instances* that motivate the pluggable design, but this intent ships only the framework + the security/dep/secret instance.
- **Forking, patching, or reimplementing any scanner.** Hard constraint. The gate wraps an unmodified upstream tool and configures it; if a wrap needs upstream behavior it lacks, that is a wrap/config problem, not a fork.
- **A Claude-Code-only delivery.** A CC skill frontend may exist as convenience later, but the local standalone CLI is the in-scope primary; shipping CC-only would violate the local-first requirement.
- **Re-hardening abcd's own pipeline.** abcd's process-protecting gates already exist; this intent targets the downstream app, not abcd's CI.
- **Sequencing relative to itd-60/61** — `/abcd:intent plan`'s job, though note the brief-change dependency below.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** a downstream app built with abcd, **when** the security validation gate runs, **then** it executes a wrapped, unmodified trusted scanner against the app and emits a per-check four-verdict result (MET / MET_WITH_CONCERNS / NOT_MET / INCONCLUSIVE).
- **Given** the configured scanner is unavailable or a trustworthy scan cannot complete, **when** the gate runs, **then** it refuses (fails closed) and never reports a silent pass.
- **Given** abcd's "never touch deps" rule, **when** the gate is implemented, **then** it wraps and configures the scanner without forking, patching, or reimplementing it.
- **Given** abcd's local-first requirement, **when** the gate is invoked outside Claude Code (CI / pre-commit / runtime hook via the standalone CLI), **then** it runs to a verdict with no Claude Code dependency.
- **Given** the pluggable framework, **when** a second risk type (e.g. a11y or privacy) is later added, **then** it plugs in as another validation discipline of the same kind without re-architecting the gate.

## Open Questions

- Which scanner(s) to wrap first for the security/dep/secret instance, and how to keep the choice swappable (the wrap must not bind abcd to one vendor)?
- Does a framework-provided validation discipline become a **baseline every abcd project inherits** (opt-out) or a configured opt-in? The brief currently says downstream apps author their own disciplines; the chosen answer is "two kinds" (framework-provided vs app-authored), and the opt-in/opt-out default is still open.
- How does the gate identify "the downstream app" to scan — by abcd project structure, an explicit target path, or configuration?
- Where does the four-verdict output feed — a new receipt type, the existing review-artifact pipeline, or acceptance evidence the fidelity reviewer consumes?
- Provenance/dependency: introducing the "validation discipline" kind edits the brief, which [[itd-61-brief-change-derivation]] would govern — does this intent wait on the doc-fidelity pair, or proceed and let them reconcile? (A `/abcd:intent plan` sequencing question.)

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._

## References

- Originating assessment: an internal comparative assessment (2026-06-26,
  recorded in the sources ledger) — names the safety pillar as abcd's largest
  gap against the amateur-coder thesis, and identifies the transplantable
  mechanisms in the companion harness.
- Governing constraints: abcd never forks/patches/reimplements a dependency
  (wrap + configure only); abcd must run locally, independent of Claude Code.
- Reuses the fail-closed, four-verdict posture of the fidelity reviewer of spc-12 (predecessor store)
  and the native ship gate (scanner-unavailable → refuse, never silent
  pass).
- Brief-change dependency: introducing framework-provided "validation
  disciplines" alongside the existing app-authored disciplines is a brief edit
  governed by [[itd-60-doc-fidelity-anti-drift]] / [[itd-61-brief-change-derivation]].
- Existing disciplines for contrast: `intents/disciplines/`
  (itd-1-acceptance-gates, itd-5-prompt-quality, itd-37-modification-grammar) —
  all app-process disciplines, none a validation gate.
