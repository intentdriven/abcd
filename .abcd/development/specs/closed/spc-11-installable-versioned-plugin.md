---
id: spc-11
slug: installable-versioned-plugin
intent: itd-67
---
# installable-versioned-plugin

## Summary

spc-11 delivers itd-67 in two slices that meet at the release cut:

- **The changelog slice** — the `abcd launch ship` write-verb, the host-delegated
  composer that writes the release prose, the completeness bijection that makes
  that prose faithful by construction, and the read-only `abcd changelog`
  preview.
- **The distribution slice** — the derived version rendered into
  `plugin.json` / `marketplace.json` **in the release payload only**, the
  documented install and update path, and a **light** installability smoke.

The version *number* itself comes from spc-10 (itd-73); spc-11 consumes it. The
programme plan is
`.abcd/development/plans/2026-07-21-derived-version-and-changelog.md`; its
**Outcomes** section is authoritative where it touches anything below.

## Scope

### Changelog slice

- **`abcd launch ship`** — the write-verb, which does not exist today: `launch`
  is dry-run-only and `Ship()` stops at `WouldPublish`. Mirroring `disembark`, it
  is a host-orchestrated flow over **two Go entry points**, not one linear call:
  1. **emit** — the deterministic record set, the derived version, and the
     guardrail result;
  2. the host runs the composer agent;
  3. **ingest** (`--changelog-json`) — run the bijection, then write the dated
     heading and render the release-payload manifest version.
- **Two fail-closed refusals**, wired into the emit step:
  - any intent in `planned/` whose spec is **closed** — it should have
    auto-moved to `shipped/` (itd-80), so the cut would silently under-bump. The
    refusal names it.
  - the newest CHANGELOG heading **ahead** of the newest tag — a release is in
    flight and the base is mismatched.
- **A new release-changelog composer agent**, distinctly named (the `CHANGELOG`
  and `README` agent slots are occupied by two documents the plugin loader
  mis-registers as agents — iss-110). Modelled on `press-release-composer` /
  `principle-distiller`: host-delegated per ADR-25, cite-or-be-dropped, an
  injection-canary fixture, and a `prompt_version` per itd-5.
- **The completeness bijection** — the set of record ids cited by the composer
  must equal the required set **minus** every `internal` record: no omission, no
  invention. A mismatch loud-stages and refuses; nothing partial is ever written.
  The agent controls **wording and the Keep-a-Changelog section only**; the
  version and the inclusion set are deterministic.
- **`abcd changelog`** (bare) — the **deterministic-only** preview: next version,
  the deciding `impact`, the record list, and the guardrail status. No agent, no
  prose, no writes. Prose is generated exactly once, at the reviewed ship.

### Distribution slice

- **`version` rendered into `plugin.json` and `marketplace.json`** in the
  **release payload only**. The dev tree stays version-**absent** (ADR-19); the
  existing `launch/lockstep.go` release-mode contract enforces the pair.
- **The install and update path in the README** —
  `/plugin marketplace add REPPL/abcd` → `/plugin install abcd@abcd-marketplace`;
  update with `/plugin update abcd`.
- **A light installability smoke** — `marketplace.json` and `plugin.json` parse,
  `source` resolves, and every declared command/skill/agent/hook path exists. A
  missing path FAILS.

## Approach

The cut stays a **reviewed pull request** (ADR-37): everything derived lands in
the ship PR, reviewed once, then durable. `auto-release.yml` and `release.yml`
are untouched — the generated heading matches the grep they already run, so the
workflow contract is preserved rather than replaced. No bot writes to `main`.

The prose is the only non-deterministic part of the cut, so it sits outside the
deterministic core and behind a verifier: the Go side computes the exact record
set, the agent writes lines that each cite a record id, and the Go side accepts
the result only if the cited set matches exactly. Faithfulness is structural, not
a matter of trusting the model. The two-entry-point shape exists for the same
reason `disembark` has it — the host, not the core, is where an agent runs, and a
CI-only context has no model. If the composer is unreachable, the verb loud-stages
and writes nothing.

`impact` never decides the Keep-a-Changelog section. The bijection guards
**id-completeness only**; a four-value enum cannot express `Security` or
`Deprecated`, so the section is the composer's judgement from record content.

The light smoke resolves the declared surface through the **same seam** itd-66's
deep tier will use, so the deep tier is a drop-in upgrade of the assertion rather
than a rewrite of the resolution.

## Migration

One clean cutover, not a fold. The maintainer performs a final manual ADR-37 roll
of the current hand-written `[Unreleased]` block, picking the version. The
derived machinery then starts pristine from the **next** cut: `[Unreleased]` is
empty and every subsequent section is fully derived. The current `[Unreleased]`
mixes shipped issues with a not-yet-shipped intent, so it does not map onto the
derived set and folding it would double-count.

## Reconciliations with the intent as written

The intent predates itd-73 and its own resolution of the open questions below;
these are recorded as decided, not left open.

- **Bump-tier selection.** itd-67's "a phase completed since the last launch →
  minor" heuristic is **superseded** by itd-73's `impact` derivation (spc-10).
  The bump falls out of the records' declared `impact`, not out of phase
  membership. spc-11 consumes the number and never computes one.
- **Where the changelog lives.** A single `CHANGELOG.md` whose newest dated
  `## [X.Y.Z] - <date>` heading `auto-release.yml` greps. Not the marketplace
  entry, not tag annotations.
- **Generated, not hand-curated.** From the shipped intents and resolved issues,
  via the composer agent under the bijection — not from the launch report and not
  from commit-message parsing.
- **Seed version.** The base is whatever the newest **git tag** says, read at the
  cut; there is no fixed seed and no fixed "next after 0.3.0".
- **Tagging.** `launch ship` does not tag. It writes the dated heading in the ship
  PR; `auto-release.yml` tags on merge. This is ADR-37 preserved, and it is what
  itd-67's "tags the repo" criterion resolves to in the current architecture.
- **Refusing an empty cut.** Nothing shipped since the tag → no bump, "nothing to
  release", no heading written. A forced patch re-snapshot is not offered.
- **Where the install path is documented.** The repo README. Not mirrored into
  `docs/`.

## itd-66 is deferred

**itd-66 (`launch-payload-render-parity`) is DEFERRED to a follow-up, after this
programme.** itd-67's own acceptance criteria frame its smoke as *light … later
upgraded to call* itd-66's deep version, so the deferral is a clean split rather
than a blocker. itd-66 owns the materialised payload render, the `.abcd/**`
leak-proof assertion, symlink resolution, the parity diff against the previous
release, and the isolated-subprocess deep smoke. spc-11 builds the light tier and
positions the surface-resolution seam so itd-66 slots in.

## Acceptance-criteria satisfaction

- **The marketplace resolves and lists `abcd`** — the light smoke parses
  `marketplace.json`, asserts the single `abcd` plugin entry, and resolves
  `source: "./"`.
- **Install registers the full `/abcd:*` surface** — the light smoke asserts every
  declared command/skill/agent/hook path exists in the payload; a missing path
  FAILS. Tested in both directions.
- **A new ship bumps `plugin.json.version`, updates `marketplace.json`, and the
  repo is tagged** — the derived version is rendered into both manifests in the
  release payload under the existing lockstep contract; the tag follows from the
  dated heading via the unchanged `auto-release.yml`.
- **The bump tier and its reason are recorded** — `abcd changelog` and the emit
  step both report the next version **and the deciding `impact`**, naming the
  record that decided it.
- **An auto-recorded changelog generated from shipped intents and resolved
  issues, with a single home** — the composer flow above, under the bijection,
  writing one `CHANGELOG.md`.
- **The version lives in the manifest and tags, not duplicated across doc
  bodies** — the dev tree carries no version; the sole in-tree carrier is the
  CHANGELOG dated heading, and the manifest fields exist only in the rendered
  release payload.

## What this spec does not deliver

- The `impact` field, the two lints, the derivation arithmetic, and the surface
  guardrail — spc-10 (itd-73).
- Everything itd-66 owns (see the deferral above).
- Publishing to any registry beyond the git repo and its tag.
- Pre-release channels.
