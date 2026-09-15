---
id: itd-93
slug: abcd-scaffolds-a-hardened-changelog-driven-release-gate-into
spec_id: spc-14
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
related_adrs: [adr-37]
severity: minor
---

# abcd Scaffolds a Release Gate That Works on the First Try

## Press Release

> _Design settled in the 2026-07-24 maintainer grill (see DECISIONS.md); the
> four resolved decisions are folded below and recorded in § Open Questions._

> **A repo abcd manages gets a release process that is correct the day it goes
> public — no self-inflicted first-release failure.** Ask abcd to set up
> releases and `abcd launch scaffold` lands a changelog-driven release gate:
> rolling `[Unreleased]` into a dated version in a reviewed PR is the release
> decision, and on merge the automation tags exactly that commit and publishes.
> The gate that verifies the release is armed against the *reviewed content
> commit*, so the very first public release cannot hit the receipt-vs-tag
> self-reference that once blocked abcd's own. Before that first release the
> operator runs a built-in rehearsal — a `workflow_dispatch` dry run that arms
> the full gate against a simulated release and publishes nothing — so a green
> rehearsal proves the gate works before it is ever trusted with a real tag.
>
> "I flipped my repo public and cut a release the same afternoon — it just
> worked," said Alice, a solo founder. "I didn't have to discover, the hard
> way, that my release gate could never be satisfied. abcd gave me the version
> abcd itself only reached after a day of untangling."

## Why This Matters

When abcd-cli went public and cut its first release, the semantic release gate
— which had never run end-to-end while the repo was private — **fail-closed by
construction**: it armed against the tagged commit and read the reviewer
receipts from that commit's own tree, but a receipt names the commit its
reviewer read and can only live in a *later* commit, so a receipt naming the
tagged commit can never sit in the tagged commit's tree. The fix
([PR #99](https://github.com/REPPL/abcd-cli/pull/99), recorded in
[iss-108]) was to arm against the reviewed *content* commit (`HEAD^2^` on the
auto-release merge path, `HEAD^` on a direct tag) and to structure the release
branch as two commits — the changelog roll, then the receipts naming it.

That flaw was abcd-cli's own CI, and it did **not** reach managed repos —
abcd ships and scaffolds no release workflow today ([iss-108] verified this:
`launch-payload.json` excludes `.github/`, ahoy/launch write no CI). But that is
exactly the gap this intent closes: abcd is a *configuration layer for
development*, and "how you cut a correct, gated release" is configuration a
managed repo should be able to inherit rather than re-derive — and re-derive
*with the same latent self-reference bug baked in*. The lesson abcd paid for
once should ship as a working default, not a trap every managed repo rediscovers
at its first public release.

## What's In Scope

- **A `launch` sub-verb** (`abcd launch scaffold`), extending the existing
  04-launch surface — which already owns how a release is cut and gated — that
  writes, into a managed repo that lacks them, the fixed release machinery: a
  `release.yml` (verify → build → publish, gate armed against the reviewed
  content commit) and an `auto-release.yml` (detect newest dated CHANGELOG
  version → tag that commit → call `release.yml`), both `GITHUB_TOKEN`-only and
  injection-safe.
- **The adr-37 policy, carried as a per-repo runbook**: the CHANGELOG is the
  release instrument; rolling `[Unreleased]` → `## [X.Y.Z] - <date>` in a
  reviewed PR is the release decision; the two-commit release-branch shape
  (roll → receipts) is documented so `HEAD^2^` resolution holds.
- **Self-scaffold parity — one template, proven by abcd-cli's own release**:
  the scaffold ships a single template, and abcd-cli's own
  `release.yml`/`auto-release.yml` are regenerated from it (with abcd-cli's
  substitutions) under a test that asserts the tree matches the template output.
  The proven pattern and the shipped template are one artifact by construction,
  so every abcd release exercises the exact machinery a managed repo receives.
- **A built-in `workflow_dispatch` rehearsal mode**: the scaffolded workflow
  carries a dry run that arms the full gate against a simulated changelog roll
  and reviewed-content commit, asserts the gate admits it, and publishes
  nothing. The runbook makes a green rehearsal the precondition for the first
  real release.
- **Producer-agnostic on the changelog seam**: the dated CHANGELOG heading *is*
  the seam with derived versioning (itd-73). `auto-release` keys on the newest
  dated heading, so `launch ship`'s derived version is one optional producer and
  a hand-rolled heading fires the same machinery; the scaffold stays
  producer-agnostic.
- **The receipt/charter interop already fixed in abcd-cli**: the sha-keyed
  receipt-dir convention plus the `check-reviews` (RD001) exemption, so the two
  in-repo review conventions do not collide.
- **Wiring to the repo's own facts**: the required-status-check contexts and the
  release-gate's required detectors are derived from the target repo's actual CI
  job names / configured gates, not hard-coded to abcd-cli's.
- **The deterministic ship flow** *(folded 2026-08-19 from iss-327, the
  maintainer's decision after abcd-cli's own v0.6.0 tag fail-closed
  unpublishably — iss-326)*: the gate that today fires at tag time — the most
  expensive, unrecoverable moment — shifts left into the verb, so the happy
  path is the only path an operator can walk. Three pieces: the **emit step
  prints the receipts protocol as its own checklist** (run the semantic gates
  against the commit this cut produces; receipts commit on top) instead of
  assuming the runbook was read; the **ingest step refuses to finish a cut it
  cannot prove releasable** — it refuses on `main`, and a cut whose content
  commit lacks validating PROMOTE receipts is loudly incomplete, never
  quietly mergeable; and a **`launch receipts` sub-verb runs the exact
  receipt-gate check the release job runs**, locally, before the merge — a
  red result costs an amend instead of a dead tag. Bootstrap ordering belongs
  to the tool, not the operator's memory.
- **Idempotent + fail-safe scaffolding**: re-running is a no-op when the
  machinery is current; it never overwrites a workflow the operator hand-edited
  without a transparent-confirm; it refuses rather than half-writing.

## What's Out of Scope

- **The semantic detectors themselves** (`docs-currency-reviewer`,
  `brief↔surface cross-check`) — they are host-run LLM passes, not scaffolded CI;
  a managed repo opts into them (or runs none) and the gate's required-detector
  list reflects that. Scaffolding must degrade cleanly to the deterministic
  gates alone when no semantic detector is configured.
- **Signing/attestation infrastructure** beyond what the built-in
  `GITHUB_TOKEN` + `actions/attest` already provide (no new dependency, no PAT).
- **Non-GitHub forges** — this scaffold targets GitHub Actions; other CI hosts
  are a later concern.
- **Choosing the version number** — that remains adr-31/itd-73 (derived from
  intent impact); this intent scaffolds the *cutting and gating* machinery.

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the [itd-1 discipline](../disciplines/itd-1-acceptance-gates.md).
> The bar was confirmed in the 2026-07-24 maintainer grill: the five seeded
> criteria stand, amended for the rehearsal-mode decision (c), with a sixth
> covering the rehearsal itself._

- **Given** a managed GitHub repo with no release workflow, **when** the
  operator runs the `launch scaffold` sub-verb, **then** `release.yml`,
  `auto-release.yml`, and the release runbook are written, wired to the repo's
  own CI check names, and pass the repo's workflow audit (e.g. zizmor) with no
  injection or duplicate-key findings.
- **Given** a repo with the scaffolded gate **and a green `workflow_dispatch`
  rehearsal on record**, **when** it cuts its first public release (roll
  `[Unreleased]` → dated heading, merge), **then** the release publishes — the
  gate is armed against the reviewed content commit and does **not** hit the
  receipt-vs-tag self-reference (a test exercises the merge path and asserts a
  published release, not a fail-closed gate).
- **Given** a repo that configures **no** semantic detector, **when** it
  releases, **then** the deterministic gates alone admit the release and the
  receipt gate requires nothing (no host-run pass is silently treated as
  missing).
- **Given** the machinery is already present and current, **when** the
  `launch scaffold` sub-verb runs again, **then** it reports a no-op and mutates
  nothing; **and given** an operator hand-edited a scaffolded workflow, **when**
  the verb runs, **then** it refuses or transparent-confirms rather than
  clobbering.
- **Given** the scaffolded `check-reviews`/RD001 charter, **when** sha-keyed
  receipt directories exist, **then** they are exempt from the dated-review-dir
  shape (the abcd-cli collision does not recur in the managed repo).
- **Given** the scaffolded rehearsal mode, **when** the operator triggers the
  `workflow_dispatch` rehearsal, **then** the full gate is armed against a
  simulated changelog roll and reviewed-content commit, the gate admits it, and
  nothing is published — a green rehearsal proves the gate before the first real
  release.

- **Given** an operator on a release branch whose content commit has no
  receipts, **when** they run the `launch receipts` check (or the ingest step
  completes), **then** the output names each missing or non-PROMOTE receipt
  and the commit it must name — and the same repository state fails the
  release job's receipt gate identically, so the local check and the remote
  gate can never disagree (folded 2026-08-19, iss-327).
- **Given** an operator who runs the emit step, **when** it renders the cut,
  **then** the output ends with the receipts protocol as a numbered checklist
  — the semantic gates to run, the commit to key receipts to, and the
  two-commit branch shape — so a first-time operator learns the protocol from
  the verb, never from a failed release run (folded 2026-08-19, iss-327).

## Prior Art

- [adr-37](../../decisions/adrs/0037-changelog-driven-releases.md) — the
  changelog-driven release policy this scaffolds.
- iss-108 — the self-reference flaw, its abcd-cli fix (PR #99), and the verified
  finding that no release machinery currently reaches managed repos.
- `.abcd/development/release-gate/` — abcd-cli's own runbook + detectors, the
  proven pattern to generalise.

## Open Questions

_All four resolved in the 2026-07-24 maintainer grill (see DECISIONS.md,
2026-07-24 entries). The PRD is synthesised from that interview; promotion is
queued in `../../plans/2026-07-24-next-run-queue.md` (Track 1)._

- **Which surface scaffolds it?** RESOLVED: a `launch` sub-verb (e.g.
  `abcd launch scaffold`) — launch already owns how a release is cut and
  gated; explicit and opt-in; extends the existing 04-launch chapter.
  Rejected: new top-level verb (new surface to reconcile), `ahoy install`
  step (release CI silently arriving with install is the non-deliberate path
  this intent argues against), embark-time record family (couples to a
  round-trip most managed repos won't use).
- **How much is templated vs. copied?** RESOLVED: **self-scaffold parity** —
  one template; abcd-cli's own `release.yml`/`auto-release.yml` are
  regenerated from it (with abcd-cli's substitutions) and a test asserts the
  tree matches the template output. The proven pattern and the shipped
  template are one artifact by construction; every abcd release exercises the
  exact machinery managed repos get. Rejected: lockstep diff test between two
  hand-maintained artifacts; frozen verbatim copy.
- **Private→public activation.** RESOLVED: **built-in rehearsal mode** — the
  scaffold ships a `workflow_dispatch` dry-run that arms the full gate
  against a simulated changelog roll and reviewed-content commit, asserts the
  gate admits it, and publishes nothing. The runbook makes a green rehearsal
  the precondition for the first real release.
- **Relationship to itd-73** (derived versioning) — RESOLVED: the CHANGELOG
  dated-heading format **is** the seam; `auto-release` keys on the newest
  dated heading, so `launch ship`'s derived version is one optional producer
  and a hand-rolled heading fires the same machinery. The scaffold stays
  producer-agnostic.

## Audit Notes

_Empty. Populated by intent-fidelity-reviewer when intent moves to shipped/._
