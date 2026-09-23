---
id: itd-85
slug: audit-verb
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
---

# A Read-Only Repo-Conformance Audit

## Press Release

> **abcd ships `abcd audit`** — a read-only check that tells you, in one command, whether a repository follows the working conventions: the three-tier `.abcd/` layout, an `AGENTS.md` router, durable decisions, current Diátaxis docs, and privacy hygiene. It runs against any repo given only a working directory, prints a grouped human report with a fix message for each gap, and emits machine JSON carrying a stable rule id and severity per finding. `abcd ahoy doctor` still answers "is the tool set up here"; `abcd audit` answers "does this repo conform" — two different questions, two verbs. Onboarding stops being guesswork: `prepare-this-repo` runs `abcd audit` and shows exactly what to fix.
>
> "I hand-built some `.abcd/` structure and thought I was done," said Alice, a maintainer. "`abcd audit` told me in a second that my `work/` tier was missing and my decisions were sitting in the gitignored layer — the two things that actually mattered."

## Why This Matters

Today `abcd ahoy` reports abcd's own install-plumbing gaps (config, rules, marker block, history store) and is silent on the *convention* gaps `prepare-this-repo` exists to fix — a missing committed `.abcd/work/` tier, an absent `AGENTS.md` router, decisions leaking into the gitignored layer, docs drift, privacy hygiene. The onboarding audit is performed by an agent reading prose, not by the binary, so onboarding is glue rather than engine (captured as `iss-86`). `abcd audit` is the read-only surface that closes that gap: the binary checks the conventions, and `prepare-this-repo` consumes its result instead of hand-auditing. Its tri-state exit code lets a repository wire it into CI, though abcd's own CI does not run its conformance check; making a narrow, high-precision subset of its findings merge-blocking is ruled (2026-09-23, iss-2608231000561060) and not yet built.

## What's In Scope

- A new read-only `abcd audit` verb, distinct from `ahoy doctor` (doctor = tool-setup health; audit = repo conformance). Wired from both front doors: CLI plus `commands/audit.md`, with a `shipped` row in the `04-surfaces` registry so `surface_coverage` stays honest.
- Engine reuse: build on `internal/core/lint` (severity + fix-message + allow-context), adding the two primitives it lacks — path presence/absence and simple structural assertions (a path is gitignored; a directory holds at least one entry).
- A declarative rule schema adapting repolinter's vocabulary: `{ id, severity (error|warn|off), where (conditional enablement), fix, policyInfo }`, separate from the evaluator, with bundled in-binary defaults.
- Five v1 rules: `three-tier-layout` (error), `conventions-router` (error), `decision-durability` (warn), `docs-currency` (warn, reusing docs-lint), `privacy-hygiene` (error).
- Native compact JSON on stdout (stable rule ids) plus a grouped human render with per-finding severity glyph and inline fix message, diagnostics routed to stderr. Conftest-style tri-state exit: `0` clean, `1` warnings only, `2` any error.
- Wiring: `prepare-this-repo` Phase 2 consumes `abcd audit --json` in place of the hand-produced gap report.

## What's Out of Scope

- SARIF export (`--format sarif`) — deferred to a later phase; native JSON ships first, behind a serializer seam that makes SARIF a thin add-on when CI/IDE ingestion is wanted.
- Repo-level rule overrides / config extension — a later phase; v1 ships bundled defaults only.
- Any auto-fix or mutation — `audit` is strictly read-only; remediation stays with `prepare-this-repo` and the maintainer.
- The `managed-repo` adoption-state check (`iss-88`) — that stays an `ahoy`/detection concern; folding it here would re-blur the audit/doctor split.
- Any external rule engine (OPA/Rego, and the archived repolinter) — rejected on the no-new-dependency hard stop.

## Acceptance Criteria

> _Required (per the itd-1 discipline). At least one Given-When-Then bullet describing the verifiable bar for "shipped"._

- **Given** a repo missing `.abcd/work/`, **when** `abcd audit --json` runs, **then** the `three-tier-layout` rule reports `severity: error` with a fix message naming the missing tier, and the process exits `2`.
- **Given** a repo whose decisions live only in the gitignored `.work.local/` layer, **when** `abcd audit` runs, **then** `decision-durability` reports `warn` and the process exits `1` (no errors present).
- **Given** a committed file containing an absolute local path, **when** `abcd audit` runs, **then** `privacy-hygiene` reports `error` citing `file:line`, **unless** a waiver escape is present on that line.
- **Given** a conforming repo, **when** `abcd audit` runs, **then** it exits `0` with a green human render, **and** `abcd audit --json` emits `{ "findings": [] }`.
- **Given** `docs/` is absent, **when** `abcd audit` runs, **then** the `docs-currency` rule is skipped via its `where` condition rather than failed.

## SOTA

> _Per the [sota-per-intent principle](../../principles/sota-per-intent.md): existing alternatives + rough maturity, then the chosen path. Harvested from the SOTA-researched design plan [`2026-07-13-abcd-audit-verb.md`](../../plans/2026-07-13-abcd-audit-verb.md)._

- **Repo-convention rule engine.** Alternatives: repolinter (declarative rulesets — *mature but archived*, Node runtime); MegaLinter (*mature*, Docker orchestrator of ~50 linters). Both are foreign runtimes for a single Go binary. repolinter's rule-object *schema* (`id/severity/where/fix/policyInfo` + `extends`) is adopted as a data model only. → **Bespoke** on `internal/core/lint`. No dependency.
- **Severity + exit-code semantics.** Alternatives: OPA Conftest (`deny`/`warn` namespaces, exit `0`/`1`/`2` — *mature*); the OPA/Rego engine is the one option genuinely embeddable in Go but is heavy and forces Rego authoring. Conftest's severity→exit vocabulary is adopted as convention; the engine is rejected. → **Adapt** behind the engine. No dependency.
- **Result interchange.** SARIF 2.1.0 (*de-facto standard*, ingested by GitHub code-scanning; emitted by checkov/MegaLinter/Conftest). Adopted as an *optional export* format behind a serializer seam, deferred to a later phase. → **Adapt**, later. No dependency.
- **Presentation.** The `doctor` UX pattern (*consensus*: grouped checks, severity glyph, inline actionable fix). → **Adopt** as human-render presentation over the engine's results.

**Verdict — bespoke-with-seam.** No new dependency ⇒ no path-1 hard stop; the seams are load-bearing (a rule-loader seam and an output-serializer seam) ⇒ no path-3 bespoke-no-seam review. Build proceeds without a dependency gate.

## Open Questions

- Repo-level rule override/config format — deferred; v1 ships bundled defaults only.
- Whether `decision-durability` graduates from `warn` to `error` once the convention is firmly established.
- **Verb placement — resolved 2026-07-13.** A new read-only `abcd audit` verb, distinct from `ahoy doctor`: doctor answers tool-setup health, audit answers repo conformance. Plan-review treats this as decided rather than re-opening.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
