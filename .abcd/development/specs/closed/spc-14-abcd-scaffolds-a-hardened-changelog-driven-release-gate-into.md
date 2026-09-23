---
id: spc-14
slug: abcd-scaffolds-a-hardened-changelog-driven-release-gate-into
intent: itd-93
---
# abcd-scaffolds-a-hardened-changelog-driven-release-gate-into

## Summary

spc-14 delivers itd-93: a `launch` sub-verb (`abcd launch scaffold`) that writes,
into a managed repo that lacks them, the fixed changelog-driven release machinery
abcd-cli reached only after untangling its own first-release failure — a
`release.yml` and an `auto-release.yml` whose verify gate is armed against the
reviewed *content* commit, so the very first public release cannot hit the
receipt-vs-tag self-reference. The scaffold ships one template that abcd-cli's own
release workflows are regenerated from (self-scaffold parity), and carries a
built-in `workflow_dispatch` rehearsal a repo runs green before its first real
release.

This spec is the product requirement synthesised from the 2026-07-24 maintainer
grill (DECISIONS.md) and the intent. Its four design decisions are settled here;
the **implementation design and build are a separate, focused run** and are out
of scope for this promotion (2026-07-24 next-run queue, Track 1: "Promotion only
— implementation is NOT in this run").

## Scope

- **Surface — a `launch` sub-verb** (`abcd launch scaffold`), extending the
  existing 04-launch chapter, which already owns how a release is cut and gated.
  It is explicit and opt-in.
- **What it writes:** into a managed repo with no release workflow, a
  `release.yml` (verify → build → publish, the verify gate armed against the
  reviewed content commit — `HEAD^2^` on the auto-release merge path, `HEAD^` on
  a direct tag) and an `auto-release.yml` (newest dated CHANGELOG version → tag
  that commit → call `release.yml`), both `GITHUB_TOKEN`-only and injection-safe,
  plus the adr-37 release runbook.
- **Self-scaffold parity — one template.** abcd-cli's own
  `release.yml`/`auto-release.yml` are regenerated from the shipped template
  (with abcd-cli's substitutions) under a test that asserts the tree matches the
  template output, so the proven pattern and the template are one artifact.
- **A built-in `workflow_dispatch` rehearsal mode** that arms the full gate
  against a simulated changelog roll and reviewed-content commit, asserts the
  gate admits it, and publishes nothing. The runbook makes a green rehearsal the
  precondition for the first real release.
- **Producer-agnostic on the changelog seam.** The dated CHANGELOG heading *is*
  the seam with derived versioning (itd-73); `auto-release` keys on the newest
  dated heading, so `launch ship`'s derived version is one optional producer and
  a hand-rolled heading fires the same machinery.
- **Receipt/charter interop** already fixed in abcd-cli: the sha-keyed
  receipt-dir convention plus the `check-reviews` (RD001) exemption, so the two
  in-repo review conventions do not collide.
- **Wiring to the repo's own facts:** required-status-check contexts and the
  release-gate's required detectors are derived from the target repo's actual CI
  job names / configured gates, not hard-coded to abcd-cli's.
- **Idempotent + fail-safe:** re-running is a no-op when the machinery is
  current; it never overwrites a hand-edited workflow without a
  transparent-confirm; it refuses rather than half-writing.

Out of scope (itd-93 § What's Out of Scope): the semantic detectors themselves
(host-run LLM passes, not scaffolded CI — the scaffold degrades cleanly to the
deterministic gates when none is configured); signing/attestation beyond the
built-in `GITHUB_TOKEN` + `actions/attest`; non-GitHub forges; and choosing the
version number (adr-31/itd-73). **Also out of scope for this spec:** the
implementation design — the concrete workflow YAML, the template-substitution
mechanism, and the verb wiring are deferred to the focused implementation run.

## Design decisions (resolved in the 2026-07-24 maintainer grill)

| # | Question | Resolution | Rejected |
|---|---|---|---|
| (a) | Which surface scaffolds it? | A `launch` sub-verb (`abcd launch scaffold`) extending 04-launch — launch already owns how a release is cut and gated; explicit and opt-in. | New top-level verb (new surface to reconcile); `ahoy install` step (release CI silently arriving with install is the non-deliberate path this intent argues against); embark-time record family (couples to a round-trip most managed repos won't use). |
| (b) | Templated vs. copied? | Self-scaffold parity: one template; abcd-cli's own workflows are regenerated from it and a test asserts the tree matches the template output, so the proven pattern and the template are one artifact. | Lockstep diff test between two hand-maintained artifacts; frozen verbatim copy. |
| (c) | Private→public activation | Built-in `workflow_dispatch` rehearsal mode: arms the full gate against a simulated roll and reviewed-content commit, asserts it admits, publishes nothing; a green rehearsal is the runbook precondition for the first real release. | — |
| (d) | Relationship to itd-73 | The dated CHANGELOG heading **is** the seam; `auto-release` keys on the newest dated heading, so `launch ship`'s derived version is one optional producer and a hand-rolled heading fires the same machinery. The scaffold stays producer-agnostic. | — |

## Approach

The design generalises abcd-cli's own release gate — its runbook and detectors
under `.abcd/development/release-gate/`, and the self-reference fix recorded in
iss-108 (PR #99) — into configuration a managed repo inherits rather than
re-derives with the same latent bug baked in.

The load-bearing correctness property is the same one abcd-cli paid for once: the
verify gate arms against the reviewed *content* commit, not the tagged commit, so
a receipt naming the release can live in a later commit than the one it names.
The scaffold carries the two-commit release-branch shape (roll → receipts) so
`HEAD^2^` resolution holds, and derives the required-status-check contexts and
required detectors from the target repo's own CI rather than abcd-cli's.

Self-scaffold parity (b) makes the shipped template and abcd-cli's proven
workflows one artifact: every abcd release exercises the exact machinery a
managed repo receives, and a parity test fails if they drift. The rehearsal mode
(c) closes the private→public activation gap that produced the original failure —
a gate that never ran while the repo was private now proves itself green, against
a simulated roll that publishes nothing, before the first real tag.

**The implementation design is deferred.** The concrete `release.yml` /
`auto-release.yml` contents, the template-and-substitution mechanism, the parity
test, the rehearsal `workflow_dispatch` job, and the `launch scaffold` verb wiring
are the focused implementation run's work (2026-07-24 queue, "itd-93
implementation — focused run once promoted"); synthesising them here would exceed
what the grill settled. This spec fixes the product requirement and the four
design decisions the implementation builds against.

## Acceptance-criteria satisfaction

Each criterion in itd-93 § Acceptance Criteria maps to the design element that
delivers it; the implementation run builds each and lands the test the criterion
names.

- **Scaffold writes wired, audit-clean workflows** — the `launch scaffold`
  sub-verb (a) writes `release.yml`, `auto-release.yml`, and the runbook, wired to
  the repo's own CI check names, `GITHUB_TOKEN`-only and injection-safe so the
  repo's workflow audit (e.g. zizmor) is clean.
- **First public release publishes after a green rehearsal** — the reviewed
  content-commit gate arming, plus the rehearsal precondition (c), together make
  the first release publish rather than fail closed on the receipt-vs-tag
  self-reference; a test exercises the merge path and asserts a published release.
- **No semantic detector configured → deterministic gates alone admit** — the
  scaffold degrades cleanly to the deterministic gates and the required-detector
  list reflects that none is configured, so no host-run pass is treated as
  missing.
- **Re-run is a no-op; hand-edits are not clobbered** — the idempotent,
  fail-safe scaffolding reports a no-op when current and refuses or
  transparent-confirms rather than overwriting a hand-edited workflow.
- **Sha-keyed receipt dirs are exempt from the dated-review-dir shape** — the
  scaffolded `check-reviews`/RD001 charter carries the exemption, so the abcd-cli
  collision does not recur in the managed repo.
- **Rehearsal arms the full gate and publishes nothing** — the built-in
  `workflow_dispatch` rehearsal (c) arms the full gate against a simulated roll
  and reviewed-content commit, admits it, and publishes nothing, proving the gate
  before the first real release.
