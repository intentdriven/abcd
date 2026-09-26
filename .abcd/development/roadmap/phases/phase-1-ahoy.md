> **Retired on 2026-09-21** (adr-2609212115255771): phases and milestones are no longer units of the record. Sequencing is dependencies plus the lifecycle shelves, rendered as the Now / Next / Later status block; the checkpoint is the derived release. This document stays as history and is not maintained.

# Phase 1 — Install and launch

## Expectation

By the end of this phase — the **first milestone** — a user can install abcd and
cut a release with it. `/abcd:ahoy install` runs in any folder: the folder is
classified (managed-repo / unmanaged-repo / unmanaged-folder), registered in
the history store's `~/.abcd/history/index.json`, given its CLAUDE.md marker
block, and wired with the modular rules loader; re-running is idempotent. `/abcd:launch` cuts a **curated
single-repo release** — a release artifact built from the one repository, with
the packaging filter denying `.abcd/**` so the design record is not carried by
the released binaries (per
[adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)).
There is no separate public repository to promote into: the repo *is* the
marketplace, and launch is the packaging-and-scrub step that produces the
distributable.

This is the **first milestone and the first phase that delivers commands a user
actually invokes** — the moment abcd becomes something a contributor installs
and releases with, rather than something that merely exists. The delivery order
is **MVP → the companion harness → Claude Code**: install-and-launch is the MVP surface that
proves the Go core, the adapter seams, and the packaging path end to end before
any deeper backend is wired.

## Milestone

- `/abcd:ahoy install`, `uninstall`, `dry-run`, and `doctor` all run on a fresh
  repo and on a re-run, per the acceptance in `04-surfaces/01-ahoy.md`.
- The folder-classification pass correctly distinguishes the folder kinds (per
  the matrix in `04-surfaces/01-ahoy.md`, which classifies `cwd` into
  `managed-repo`, `unmanaged-repo`, and `unmanaged-folder` along the
  strong-marker and `.git/` axes) and registers the managed repo in
  `~/.abcd/history/index.json`.
- The rules loader is live: keyword-triggered rule injection works; an
  unrelated prompt injects zero abcd rules.
- The oracle resolves through its **host-delegated native default** (per
  [adr-25](../../decisions/adrs/0025-host-delegated-llm-default.md)); any
  configured RepoPrompt or codex adapter is used when present, but neither is
  required for install or launch to run.
- `/abcd:launch` cuts a curated single-repo release: the packaging step excludes
  `.abcd/**` from the artifact and runs the secret/PII scan before the
  distributable is produced, per `04-surfaces/04-launch.md` and
  [adr-28](../../decisions/adrs/0028-single-repo-curated-release.md).
- The probe-only stubs for the other surfaces exist (`/abcd:disembark probe`,
  `/abcd:embark probe`, bare `/abcd:intent`, bare `/abcd:capture`) — they render
  or report, they do not yet act. Bare `/abcd:capture` (not `list`) per SD001: a
  `list` sub-verb would duplicate what the bare invocation already renders.

## Phase Acceptance

> _Roll-up acceptance per [adr-9 amendment](../../decisions/adrs/0009-phase-as-product-layer.md). Each bullet asserts an emergent, cross-intent truth or a phase-spanning user journey — never a copy of an intent's own `## Acceptance Criteria`._

- **Given** a fresh repo with abcd never installed, **when** a user runs
  `/abcd:ahoy install`, **then** in one command the folder is classified,
  registered in `~/.abcd/history/index.json`, given its CLAUDE.md marker block,
  and wired with the rules loader — a journey spanning itd-40, itd-3, and the
  ahoy command that no single intent delivers alone.
- **Given** an installed repo, **when** a user runs `/abcd:launch`, **then** a
  curated release artifact is produced from the single repository with `.abcd/**`
  excluded and the secret/PII scan passed — the whole install→release arc is
  walkable without a second repository (per adr-28).
- **Given** abcd installed in a repo, **when** any command issues an oracle
  call, **then** the call resolves through the host-delegated default and never
  hard-fails for lack of an external oracle backend — the property adr-25
  guarantees, owned by no single adapter.
- **Given** an installed repo, **when** a user's prompt contains a domain
  keyword, **then** the rules loader injects exactly the matching domain rules
  and nothing else — the just-in-time discipline-loading property that itd-3
  delivers and ahoy's marker block makes live.

## Scope

**Intents:** itd-3 (modular rules loader), itd-40 (folder classification +
the history-store `index.json` registry — and the per-repo history-store
*scaffolding* ahoy provisions per managed repo, see below), itd-2
(host-delegated oracle default — the
always-available bottom of the oracle seam).

**History-store scaffolding folds into itd-40.** ahoy provisions the per-repo
native history store (`~/.abcd/history/` keyed on root-commit SHA, `index.json`)
as part of what it sets up for each managed repo. itd-40 already owns the
managed-repo model — folder classification and the `~/.abcd/history/index.json`
registry — so "what ahoy provisions for each managed repo" is the same intent.
The store's **capture and read behaviour** is Phase 2 (the native transcript
corpus, per [adr-29](../../decisions/adrs/0029-native-transcript-corpus.md));
Phase 1 only lays down the directory scaffolding.

**Launch is the release surface.** `/abcd:launch` cuts a curated single-repo
release: the scrub-and-package step that excludes `.abcd/**` and scans for
secrets/PII before producing the distributable. There is no dev→public mirror
and no sibling repository (per adr-28); the RepoPrompt-workspace portability
case (itd-7) is deferred to Phase 6's lifeboat round-trip.

**`/abcd:capture` moved out.** The capture surface (itd-4) is Phase 2's — capture
is a distinct user-capability moment (a fast issue ledger), and it now sits with
the native history and memory stores it shares a substrate with.

**Brief plumbing-phases:** the brief's `/abcd:ahoy` end-to-end flow (Steps 0–12
of `04-surfaces/01-ahoy.md`) and the launch flow (`04-surfaces/04-launch.md`),
plus the probe-only stubs for the other surfaces.

## Maps against

- **Brief:** `04-surfaces/01-ahoy.md` and `04-surfaces/04-launch.md` (the
  commands being built); `06-delivery/01-build-sequence.md`;
  `05-internals/03-configuration.md` (rules-loader config, the history
  `index.json` registry, visibility-driven gitignore).
- **Intents deliver the expectation:** itd-3 delivers the marker block ahoy
  installs; itd-40 delivers the classification ahoy runs first and the history
  scaffolding; itd-2 delivers the host-delegated oracle default.
- **ADRs realised:** adr-3 (directory-as-truth — the lifecycle model the later
  capture and intent phases follow); adr-25 (host-delegated oracle default);
  adr-28 (single-repo curated release).

## Dependency rationale

- **itd-3 and itd-40 before `ahoy install`** — ahoy *installs* itd-3's marker
  block and *reads* itd-40's folder classification, resolved against the
  history `index.json` registry. ahoy must ship with both already in hand, so
  they precede the command flow within this phase.
- **launch after install** — launch packages an installed repo; the install
  flow and the `.abcd/**` layout it lays down must exist before packaging can
  exclude them.
- **itd-2 is independent of ahoy** — the host-delegated oracle default can be
  wired in parallel with the install flow; it is grouped here so the oracle seam
  is whole before later phases dispatch reviews and audits.
- **This phase runs after Phase 0** — every spec here inherits the Phase 0
  disciplines, plugs into the Phase 0 adapter seams, and is written in the
  settled vocabulary (itd-43). The CLI is the first front door; hooks are a
  later surface.

## Open questions

- The history-store *capture* behaviour (native transcript corpus) is Phase 2,
  not here — confirm the Phase 1 scaffolding lays down exactly the directory
  shape Phase 2's capture writes into, so no re-scaffolding is needed.
- Confirm the launch scrub stack's secret/PII gate is exercised by a test on a
  fixture repo, not just specified, before Phase 1 closes.
