# iss-35 record-lint graduation — design options (STOP for sign-off)

**Status:** SIGNED OFF 2026-07-11. The maintainer chose **Option C (hybrid)**:
build the structural `surface_coverage` record-lint rule *and* wire the LLM
cross-check as a standing release gate. The two blockers were also resolved:
docs/history/version are **user-facing surfaces** (each gets a `04-surfaces/`
chapter + README row), and consult/ingest/prepare-this-repo **reclassify from
skills to commands** via relabel (host-delegated markdown workflows, no Go
verbs; the read-only skill boundary rule is kept — the classification gave).
Implementation order: enumerate the three chapters + reclassify the three
skills (so the coverage rule can go green), then build the rule (TDD, branch),
then wire the detector gate. See [dcn-2607110000000002](../../work/decisions/dcn-2607110000000002-iss-35-graduation-signed-off-maintainer.md) and NEXT.md tasks.

The original design analysis is retained below as the rationale of record.

---

**Original framing (design gate, pre-sign-off):** the autonomous run reached
this point after the iss-35 reconciliation (batches 1–2 + the detector re-run,
150 → 24 → all non-adjudication leftovers fixed) and stopped rather than
implement, because graduating the cross-check to a record-lint rule was a design
decision with multiple credible shapes and a hard dependency on two open
adjudications.

## What "graduate the check" was meant to mean

iss-35's detector is the bidirectional brief↔surface cross-check
("spec-moves-with-the-surface"): every real surface (`commands/`, `skills/`)
resolves to a brief surface row, and every brief surface row resolves to a
shipped or explicitly-staged surface. It ran twice as an LLM workflow (22
checker agents) and drove discrepancies 150 → 24. The open task was to turn it
into a standing, cheap, deterministic gate so drift can't silently return.

## The core finding that forces a decision

**The detector is bidirectional, but only one direction is deterministically
lintable.**

- **Direction B (reality → brief), structural.** Enumerate `commands/abcd/*.md`
  and `skills/*/SKILL.md`; assert each has a brief home (a README table row or a
  surface chapter). This is deterministic — it is essentially `directory_coverage`
  (`internal/core/lint/lint.go:340`) pointed at the surface tree instead of a
  doc tree.
- **Direction A (brief → reality), semantic.** The bulk of the findings
  (false-claim 77, fictional-layout 29, stale-count 5 across both runs) assert
  that a brief claim about a *shipped* surface matches *binary behaviour* —
  flags, sub-verbs, exit codes, schema field names, `isDir` counts. Verifying
  these requires running the binary and reading Go source. A record-lint rule
  cannot check them without hard-coding binary facts into the linter, which then
  drift exactly like the docs do. **This direction is irreducibly an
  LLM/agent job** (a `docs-currency-reviewer`-style pass), not a structural lint.

So "graduate to a record-lint rule" cannot mean "port the whole detector." It
can only mean "extract the deterministic half; keep the semantic half as an
agent/periodic check." That reshaping is the decision.

## Two hard blockers (both are your open adjudications)

1. **docs/history/version have no brief surface row.** A Direction-B coverage
   rule fires on them the instant it is armed — which is exactly adjudication
   item 5 (are `docs`/`history` user-facing surface chapters, or operator-internal
   tooling that should be exempt?). The rule cannot go green until this is
   decided: add three chapters, or configure the three verbs as
   operator-internal `exempt`.
2. **Skill classification (adjudication item 6).** Whether `consult`/`ingest`/
   `prepare-this-repo` are correctly skills (the read-only boundary rule says
   they should be commands) determines whether `skills/` entries are covered by
   the same rule or a different one.

Neither is mine to decide (both are in the maintainer adjudication queue,
NEXT.md). Until they are, no Direction-B rule can be armed to a clean green.

## Architectural note

record-lint `roots` is `.abcd/development` only (`.abcd/record-lint.json`).
`commands/` and `skills/` are **outside** the lint roots — every existing rule
operates within roots. A surface-coverage rule is a new shape: it reads the
plugin surface (outside roots) and cross-checks the brief (inside roots). Doable
(the walker is generic), but it is not a config-only addition to an existing rule.

## Options

**Option A — structural coverage rule (Direction B only). Recommended.**
A new `surface_coverage` rule: enumerate `commands/abcd/*.md` + `skills/*/` and
assert each resolves to a brief surface row; optionally assert each brief row
that claims *shipped* status maps to a real surface file. Deterministic, cheap,
CI-friendly. Keep the LLM cross-check as a periodic/release-gate check for
Direction A (semantic drift). Requires: the docs/history decision (blocker 1) and
a decided **staged-vs-shipped marker convention** (below). Effort: ~1 rule + TDD,
one slice once unblocked.

**Option B — keep the LLM detector as the canonical check; operationalize it.**
Don't build a lint rule at all; make the workflow re-runnable on demand / at
release (it already is — `iss35-crosscheck.js`), and treat "graduation" as wiring
it into the release gate alongside `docs-currency-reviewer`. Catches both
directions, no false-green risk from the adjudications, but is not cheap/CI-fast
and stays non-deterministic (the re-run proved it samples a different subset each
time — it does not converge).

**Option C — hybrid (Option A + Option B).** Structural `surface_coverage` lint
in CI for fast deterministic coverage; retain the LLM cross-check as a
release-gate semantic pass. Most faithful to the bidirectional intent; most
work; still needs blocker 1 resolved for the lint half to go green.

## Sub-decision either A or C needs: the staged marker convention

Direction A's "explicitly staged, therefore not a discrepancy" is currently
**prose** ("design target, not shipped: …"). For any deterministic rule to tell a
staged row from a false shipped-claim, staging needs a **machine-detectable
marker** — e.g. a leading token (`[staged]`), a table column, or a frontmatter
list of shipped-vs-staged verbs. This is a small schema decision, but it is a
decision (it touches every surface chapter).

## Recommendation

**Option A**, once blocker 1 (docs/history taxonomy) is decided and a staged
marker convention is chosen. It captures the deterministically-checkable
invariant cheaply and leaves the semantic half to the agent tier where it
belongs. Option C is the fuller answer if you want the LLM pass wired as a
standing release gate too; it is a superset of A and can follow A.

## What is NOT blocked (for whoever resumes)

The iss-35 *reconciliation* is done to the limit of what is decidable without
you: batches 1–2 + the ratchet fixes are committed; the only remaining iss-35
discrepancies are the two adjudication clusters. iss-35 stays open only for this
graduation gate. The run's next queue item (iss-30 memory-ingest HTTP status,
iss-31 launch dogfood gate — code, failing-test-first) is independent of this
gate and can be picked up when you restart the loop or redirect it.
