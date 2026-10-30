# Derived versioning + auto-generated changelog + plugin distribution

_Plan / design artefact — dated 2026-07-21. Present tense; British English.
Personas are Alice, Bob, Carol only._

> **Status: grilled + adversarially reviewed 2026-07-21.** A maintainer grilling
> walked the full design tree, then an adversarial review broke the draft against
> the code (findings B1–B2, M1–M6, m1–m3 folded in below). The **Outcomes** section
> is authoritative and supersedes the open questions and any body text it touches.
> Nothing here is built yet. The `abcd launch ship` write-verb it depends on does
> **not** exist yet — building it is outcome 10 (Phase 3's first task).

## Outcomes — 2026-07-21 (authoritative: grilling + adversarial-review fixes; supersede the body)

1. **Shipped-set AND version base both anchored on the git TAG (supersedes RQ-3
   and §3; fixes M1).** Resolve the last release as the **newest git tag**
   `vX.Y.Z` — the only immutable anchor. Do **not** read the base from the newest
   CHANGELOG heading: `auto-release.yml` creates the tag *after* the ship PR
   merges, so in the post-merge/pre-tag window the heading is ahead of the tag and
   the two describe different releases. Compute the release set as the
   set-difference of `git ls-tree <tag> -- .abcd/development/intents/shipped/ .abcd/work/issues/resolved/`
   versus `git ls-tree HEAD -- <same>` (end-states, immune to squash/rebase). If
   the newest CHANGELOG heading is **ahead** of the newest tag (a release in
   flight), `launch ship` / `abcd changelog` **refuse/no-op** ("release <v> in
   flight — tag pending") rather than derive against a mismatched base.
   *Caveats to pin in tests (m3):* the diff is **not** rename-immune — a
   within-`shipped/` slug rename reads as delete+add; an intent **moved out** of
   `shipped/` (supersession) must emit a `Removed`/`Changed` line, not silently
   drop; the no-tag case is undefined here but does not arise (v0.1.0–0.3.0 exist).
2. **Prose is faithful-by-construction via a completeness bijection.** The
   deterministic Go core computes the exact record-id set of the cut; the agent
   writes prose with **each line citing its record id**; the Go side verifies the
   set of cited ids **==** the required set (no omission, no invention) before
   writing. Loud-stage and refuse on mismatch. The agent controls wording only.
3. **The agent judges the Keep-a-Changelog section; `impact` is version-only.**
   `impact` (`additive|breaking|fix`) drives **only** the SemVer bump. The section
   (Added/Changed/Removed/Deprecated/Fixed/Security) is the agent's judgement from
   record content; the bijection guards **id-completeness only**, not the section
   (a 3-value `impact` cannot express KaC granularity, e.g. Security/Removed).
4. **The version is release-payload-only (corrects §7 + Phase 4).** The dev-tree
   `plugin.json` stays version-**absent** (ADR-19, lockstep dev-mode). The derived
   version is written into `plugin.json` / `marketplace.json` **only in the
   rendered release payload** at ship. The sole in-tree version carrier is the
   CHANGELOG dated heading (ADR-37). **The base version is whatever the
   clean-cutover manual roll (outcome 7) sets, read from the newest git tag —
   there is no fixed "next-after-0.3.0" (fixes B2).** The repo is at `v0.3.0`; the
   manual roll advances it (e.g. to `v0.4.0`), and the first *derived* cut is
   next-after-**that** tag. (The earlier `0.1.0`/`0.2.0` figures were wrong.)
5. **Surface-break taxonomy.** Breaks (require a `breaking` intent in the cut): a
   removed or renamed command; a removed or renamed flag; a previously
   optional/absent flag becoming **required**; a removed declared manifest surface
   entry. **Not** breaks: a new command or new **optional** flag; changed
   help/description; reordering. **The snapshot is a NEW structured JSON emitter
   (fixes m2):** `GenerateReference` emits *Markdown*, **excludes the hidden
   `hook` subtree**, and carries no structured per-flag required-ness or manifest
   surface — so what is reused is the shared `NewRootCommand()` tree (one-canonical
   *root command*), not the markdown walker; the emitter adds structured flag data
   + a manifest-surface snapshot. `surface.json` is committed and **gated by a
   drift test** (as `reference_test.go` gates `commands.md`). **No-baseline is
   fail-closed (fixes M6):** there is no `surface.json` at v0.3.0, so Phase 2's PR
   **seeds the baseline** at the current release commit (v0.4.0 after the manual
   roll) and a cut with no committed baseline **refuses** rather than silently
   passing the first, highest-risk cut.
6. **Preview is deterministic-only.** `abcd changelog` (bare) renders the next
   version + the deciding `impact` + the record list + the guardrail status. **No
   agent, no prose** — prose is generated exactly once, at the reviewed ship.
7. **Clean-cutover migration (supersedes RQ-5).** Do ONE final manual ADR-37 roll
   of the current hand-written `[Unreleased]` (maintainer picks the version —
   likely 0.4.0 given the `intent ready` Added feature). The derived machinery then
   starts **pristine from the NEXT cut**: empty `[Unreleased]`, fully derived. No
   fold, no double-coverage (the current `[Unreleased]` mixes shipped issues with a
   not-yet-shipped intent, so it does not map to the derived set).
8. **`impact` is explicit on intents AND issues, with an `internal` class (fixes
   M3, M4).** Enum **`additive | breaking | fix | internal`**, set explicitly —
   **no silent default.** The old "issues default `fix`" was wrong: it
   under-bumped genuinely feature-adding issues (iss-57 added a `--source` value,
   iss-47 added the generated CLI reference — the maintainers filed both under
   `### Added`), and the surface guardrail only backstops *breaks*, never additive
   under-bumps. And a hard-defaulting bijection would force internal issues
   (iss-28/iss-82/iss-97/iss-106/iss-109 — TOCTOU, atomic-write, lint internals)
   into a user-facing changelog. Rules: **`internal` = excluded from the changelog
   and drives no bump**; the completeness bijection (outcome 2) requires
   `cited == required MINUS internal`. **Intents are never `internal`** (a
   press-release-first intent is user-facing by definition); `intent_impact_valid`
   blocks a move to `shipped/` without a valid non-`internal` impact, and a new
   `issue_impact_valid` lint blocks *resolving* an issue without a valid `impact`.
   **Back-fill once:** the ~5 historical shipped intents AND the ~17 issues
   resolved since v0.3.0 (agent-proposed, maintainer-confirmed).
9. **Host-delegation seam = mirror `disembark` (corrects §5.5 + Phase 3; fixes
   m1).** Phase 3 authors a new release-changelog composer agent modelled on
   `press-release-composer` / `principle-distiller` (host-delegated per ADR-25,
   cite-or-be-dropped = the bijection, injection-canary fixture, `prompt_version`
   per itd-5). **Name it distinctly (e.g. `changelog-composer`)** — the
   `CHANGELOG`/`README` agent *names are already taken* by two docs the plugin
   loader mis-registers as agents (`agents/CHANGELOG.md` = the itd-5 prompt-version
   log; `agents/README.md`), a latent defect filed as **iss-110** (the earlier
   "there is no `abcd:CHANGELOG` agent" was imprecise: the slot is occupied by a
   non-agent). The Go verb ingests the agent's `--changelog-json` and runs the
   bijection, exactly like `disembark … --principles-json`.

10. **Build the `abcd launch ship` write-verb + its orchestrating command (fixes
    B1, M5).** `launch` is **dry-run-only today** — `Ship()` stops at
    `WouldPublish`, `launch.md` is read-only, there is no `ship` subcommand and no
    write path. The trigger the whole plan hangs on **does not exist**; building it
    is an explicit deliverable at the **start of Phase 3**. Mirroring `disembark`,
    it is a plugin-orchestrated flow (a markdown command) over **two Go entry
    points**, not one linear call: (a) an **emit** step that produces the
    deterministic record-set + derived version + guardrail result; (b) the host
    runs the `changelog-composer` agent; (c) an **ingest** step
    (`--changelog-json`) that runs the bijection and writes the dated heading +
    renders the release-payload manifest version. `abcd changelog` (bare, preview)
    is the deterministic (a) alone, to stdout.

11. **The cut refuses if a merged feature's intent is still in `planned/` (fixes
    M2).** Derivation reads `shipped/`, but a feature's code can merge before its
    intent record moves to `shipped/` — **itd-94 is exactly this** (merged,
    user-facing `abcd intent ready`, yet its record sits in `planned/`, invisible
    to the tree-diff → silent under-bump). `launch ship` **fail-closed refuses**
    if any `planned/` intent has a **closed spec** (it should have auto-moved to
    `shipped/` per itd-80), naming it — so the ship is corrected before the cut,
    not under-bumped after it.

---

This programme covers three intents that share one seam — the release cut:

- **itd-73** (`derived-versioning`, draft) — the `impact` field, version
  derivation, the surface-diff guardrail, the impact-required lint.
- **itd-67** (`installable-versioned-plugin`, planned) — its **changelog-generation
  slice** and its **distribution slice** (manifest `version`, install/update
  path, a light installability smoke).
- **itd-66** (`launch-payload-render-parity`, planned) — the deep render +
  parity-diff + deep-smoke. **DEFERRED as a follow-up** (see Scope boundary).

---

## Context

### The problem

Two gaps meet at the same place — the release cut.

1. **Parallel-PR CHANGELOG conflicts.** Today `CHANGELOG.md` carries one rolling
   hand-written `## [Unreleased]` section (currently ~40 lines across `Fixed` /
   `Changed` / `Added`). Every PR that wants a changelog line edits that one
   block, so concurrent PRs conflict on it — the exact friction iss-24 records
   (severity `minor`, category `future-work-seed`): _"concurrent PRs never
   conflict on the Unreleased block."_

2. **No version story.** itd-73's press release states it plainly: a version is
   _"a fact about what changed"_, not a number a human picks. Right now nobody
   derives it — `plugin.json` carries **no `version` field at all** (confirmed:
   `.claude-plugin/plugin.json` has `name`/`description`/… but no `version`),
   and the version only ever appears at build time, stamped onto the binary from
   the git tag (`make build VERSION=<tag>`, per ADR-19/ADR-20/ADR-31).

### What exists today (ground truth, not aspiration)

- **ADR-37 (`changelog-driven-releases`, accepted 2026-07-17)** is the governing
  policy and is **PRESERVED by this programme, not superseded.** Its core:
  rolling the accumulated `## [Unreleased]` into a dated `## [X.Y.Z] - <date>`
  heading **in an ordinary reviewed PR _is_ the release decision.** ADR-37 §5
  already anticipates itd-73: _"When itd-73 lands, its derived number feeds the
  roll; the changelog mechanism here is the recording and cutting instrument
  either way."_ This programme _is_ that landing.

- **The sole CHANGELOG consumer is `.github/workflows/auto-release.yml`.** On
  every push to `main` it greps the newest dated heading:
  `grep -m1 -E '^## \[v?[0-9]+\.[0-9]+\.[0-9]+\] - '`, `sed`s out the version,
  and — if that version has no git tag — creates an annotated `vX.Y.Z` tag at
  that commit and invokes `release.yml` as a reusable workflow. **A push with no
  new dated heading is a no-op.** `release.yml` reads the tag, gates the commit,
  cross-compiles, and `gh release create … --generate-notes` (PR list since the
  previous tag; the CHANGELOG stays the durable record).

- **Reusable primitives already in the tree** (one-canonical-primitive says
  extend these, do not add a third copy):
  - `internal/core/launch/semver.go` — strict SemVer 2.0.0 parse/compare,
    `Semver{Major,Minor,Patch}`, `.Line()`, `.Tag()` (`v`-prefixed),
    `coreLess`/`coreGreater`. **The version arithmetic is already here.**
  - `internal/core/launch/lockstep.go` — enforces version/changelog lockstep
    between `plugin.json` and `marketplace.json` (`/plugins/0/changelog/version`
    pointer); dev-tree keys must be ABSENT, release keys present.
  - `internal/core/launch/ship.go`, `dryrun.go`, `includes.go`,
    `bundle.go` — the launch surface.
  - `internal/core/lint/` — record-lint host, incl. `validateIDUnique` and the
    id/status scanning that already reads the intent folders and issue ledger.
  - `.claude-plugin/marketplace.json` + `plugin.json` **already exist** with
    `source: "./"` — the distribution scaffolding is half-built; what is missing
    is the `version` field, the install/update docs, and the smoke.

- **Status is directory-as-truth.** Intents have **no `status` frontmatter
  field**: `drafts/` → `planned/` → `shipped/` (and `disciplines/`,
  `superseded/`) _is_ the state. Intents **do not carry `impact` yet** — this
  programme adds it. Issues carry `severity` and live in `open/` → `resolved/` →
  `wontfix/`.

---

## Goal

Ship a **fully derived release cut**: at `/abcd:launch ship` (a reviewed PR),
abcd (a) derives the SemVer from the `impact` of what shipped since the last
release, (b) generates the changelog prose for that version via a host-delegated
agent, (c) writes the dated `## [X.Y.Z] - <date>` heading into the ship PR so the
existing `auto-release.yml` grep tags it unchanged, (d) guards against a
mislabelled compatibility break, and (e) makes abcd a genuinely installable,
versioned plugin. No human ever types a version or writes changelog prose.

---

## The locked decisions (ruled in; recorded as decided)

1. **The changelog is fully derived, agent-generated prose.** No human ever
   writes changelog prose. At generation time a **host-delegated agent (ADR-25)**
   reads each shipped intent's press-release/AC and each resolved issue's
   title/body (plus the commits) and writes the prose. The human sets only the
   one-word `impact`.

2. **The version is derived** from the highest `impact` of what shipped since the
   last release: `breaking` → major, `additive` → minor, `fix` → patch.
   **Pre-1.0 policy:** while at `0.x`, `breaking` → **minor** (`0.3.x` → `0.4.0`),
   `additive`/`fix` → patch. The first `1.0.0` is a deliberate explicit
   `--version` override. (This matches ADR-37's "pre-1.0 minor may break" and its
   `Breaking` call-out.)

3. **The surface-diff guardrail** (itd-73): snapshot the `/abcd:*`
   command/flag/manifest surface; a removed or changed surface with **no
   `breaking` intent in the release FAILS the launch** — a mislabel cannot ship a
   compatibility lie. Plus an **impact-required lint**: every intent carries a
   valid `impact` (`additive`|`breaking`|`fix`) before it can ship, else
   `internal/core/lint` blocks it.

4. **The trigger is `/abcd:launch ship` — a reviewed PR — automatically.** Both
   version derivation and changelog generation run at ship. **No bot commits to
   `main` on every merge.** A read-only `abcd changelog` (bare verb) renders the
   pending preview from the records anytime, committing nothing. **ADR-37 is
   preserved** — the reviewed ship IS the release decision; no superseding ADR is
   needed. (Why the rejected alternative is rejected: see below.)

5. **Scope of this programme** = itd-73 (impact + derivation + guardrail + lint)
   + itd-67's changelog-generation slice + itd-67's distribution slice
   (`marketplace.json` + `plugin.json` version + README install/update path + a
   **light** smoke: manifest parses, `source` resolves, declared
   command/skill/agent/hook paths exist). **itd-66 (deep render + parity-diff +
   deep-smoke) is DEFERRED** — itd-67's own AC calls its smoke _"light… later
   upgraded to call"_ itd-66's deep version, so this is a clean deferral, not a
   blocker.

6. **The CHANGELOG stays a single `CHANGELOG.md`** whose newest dated
   `## [X.Y.Z] - <date>` heading `auto-release.yml` greps. Generation writes that
   heading in the ship PR, so **the workflow contract is untouched — zero
   workflow changes.**

### Why the bot-on-main alternative is rejected (recorded, not open)

Tagging or changelog-committing from CI on every merge to `main` would **reverse
ADR-37** — it moves the release decision out of a reviewed PR and into automation
that writes to `main`. ADR-37 explicitly rejected _"tagging from CI on every
merge (continuous release)"_ because abcd's releases are **curated cuts (ADR-28)**
and most merges are not releases. It also rejected _"a release bot with a PAT."_
The reviewed ship is the human decision point; the automation only _follows the
record_ (the dated heading), never authors it on `main`. This programme keeps that
invariant: everything derived lands **in the ship PR**, reviewed once, then
durable.

---

## Design in detail

### 1. The source model — where the truth lives

The release cut reads from records that already exist, plus one new field:

| Input | Where it lives today | Change |
| --- | --- | --- |
| Shipped intents | `.abcd/development/intents/shipped/itd-N-*.md` | + `impact` frontmatter |
| Resolved issues | `.abcd/work/issues/resolved/iss-N-*.md` | `severity` → default `impact: fix` |
| Commits since last release | git | read via `gitutil`, tag-anchored |
| Last release version | newest dated `## [X.Y.Z]` heading + git tag | read, not written by hand |
| The public surface | `/abcd:*` commands, flags, manifests | new snapshot artefact |

No new store. Directory-as-truth stays the lifecycle authority; `impact` is a
_property_ of an intent, never its status.

### 2. The `impact` field + the impact-required lint

- **New frontmatter field `impact` on every intent**, enum
  `additive | breaking | fix`. It is a **product judgement set once when the
  intent is shaped**, not a version. (itd-73's press release: _"Each intent
  declares one thing about itself — whether it adds, breaks, or fixes."_)
- **`internal/core/lint` gains an `intent_impact_valid` rule:** an intent whose
  `impact` is absent or not one of the three values is a **blocker**. This
  satisfies itd-73's AC: _"every intent carries a valid impact before it can
  ship."_ The rule reuses the existing intent-folder scanner in the lint host
  (same walk that `validateIDUnique` uses), not a new traversal.
- **Bundles** (itd-73 open Q, resolved below): each member declares its own
  `impact`; the **bundle's impact is the max** of its members. A newly-enforced
  discipline that breaks consumers counts as `breaking`.
- **Migration:** existing shipped intents predate `impact`. The lint blocks
  _shipping_ without it; already-shipped intents are back-filled once as part of
  Phase 1 (a bounded, enumerable set), not treated as a rolling debt. The lint's
  scope is "an intent may not move to `shipped/` without a valid `impact`."

### 3. Version derivation (deterministic, Go)

Pure function over the shipped-since-last-release set. Reuses
`launch/semver.go`:

```
prev  := parse newest dated CHANGELOG heading (== newest git tag)   // e.g. 0.3.2
bump  := max impact over {shipped intents ∪ resolved issues} since prev
next  := deriveNext(prev, bump)
```

`deriveNext` — the whole policy, testable in isolation:

| `prev` line | highest `impact` | `next` (≥ 1.0) | `next` (0.x, pre-1.0) |
| --- | --- | --- | --- |
| any | `breaking` | major++, minor=patch=0 | **minor++, patch=0** |
| any | `additive` | minor++, patch=0 | patch++ |
| any | `fix` | patch++ | patch++ |

- **The first `1.0.0` is an explicit `--version 1.0.0` override** — never
  derived. Pre-1.0 `breaking` deliberately stays a minor bump, matching ADR-37
  ("pre-1.0: a minor may break, called out under **Breaking**").
- **Empty set** (nothing shipped since the tag) → **no bump**; `launch ship`
  reports "nothing to release" and writes no heading (consistent with
  `auto-release.yml` being a no-op when there is no new dated heading). Whether a
  forced patch re-snapshot is ever allowed is itd-67 open Q — deferred; default is
  refuse-empty.

### 4. The surface-diff guardrail

- **Snapshot artefact:** a committed, machine-generated file (proposed
  `.abcd/development/release/surface.json`) enumerating the `/abcd:*` command
  set, each command's flags, and the manifest surface (`plugin.json` /
  `marketplace.json` shape). It is generated deterministically by walking the
  **Cobra command tree** — the same tree the existing `cli.GenerateReference`
  walker already walks for `docs/reference/cli/commands.md` (reuse, do not add a
  second walker).
- **At ship:** re-snapshot the current surface, diff against the last released
  snapshot. **A removed or changed surface entry with no `breaking` intent in the
  release FAILS the launch**, naming the changed surface (itd-73 AC). This catches
  the mislabel: an author marks a command-removal `additive`, the guardrail
  blocks it.
- **Scope:** structural (surface) breaks only. Behavioural breaks behind an
  unchanged surface remain the author's `breaking` judgement (itd-73's own
  leaning — not something a structural diff can see).

### 5. The ship-time generation flow (step by step)

Alice runs `/abcd:launch ship` on a reviewed release PR. Deterministically, in
order:

1. **Lint gate.** `intent_impact_valid` over every intent in the release; any
   missing/invalid `impact` aborts before anything is derived.
2. **Assemble the shipped set** (tag-anchored — §"Resolved questions").
3. **Derive the version** (§3). Report the number and the deciding `impact`.
4. **Surface-diff guardrail** (§4). A structural break with no `breaking` intent
   aborts, naming the surface.
5. **Generate the prose** — **host-delegated to an agent (ADR-25).** The agent
   reads each shipped intent's press-release/AC and each resolved issue's
   title/body + commits, and writes Keep-a-Changelog sections
   (`Added`/`Changed`/`Fixed`/`Breaking`) for this version. The Go side **passes
   the assembled record set to the agent and runs the completeness bijection**
   (grilling outcome 2: every cited record id present, none invented) before
   accepting it. The delegated worker is a **new release-changelog composer agent**
   (grilling outcome 9 — there is no `abcd:CHANGELOG` agent; `agents/CHANGELOG.md`
   is the itd-5 prompt-version log), modelled on `press-release-composer` /
   `principle-distiller`, ingested via `--changelog-json` like `disembark`.
6. **Fold + write.** The generated sections, folded together with the existing
   hand-written `[Unreleased]` body (§"Migration"), are written as a dated
   `## [X.Y.Z] - <date>` heading; `[Unreleased]` resets to empty. `plugin.json`
   and `marketplace.json` versions are set in lockstep (`launch/lockstep.go`
   already enforces the pair). All of this lands **in the ship PR** — reviewed
   once, then committed and durable.
7. **On merge**, `auto-release.yml` greps the new dated heading (unchanged
   contract) and tags → `release.yml` publishes.

Loud-staging: if any step cannot complete (e.g. the agent is unavailable in a
CI-only context), `launch ship` **fails loudly** and writes nothing partial — a
half-generated changelog never lands.

### 6. The `abcd changelog` preview verb (read-only)

`abcd changelog` (bare) renders the **pending** version + sections **from the
records, committing nothing**. It runs the same deterministic derivation (§3) and
(optionally) the same host-delegated prose pass, but to stdout only. This is the
zero-write status render (mirrors `abcd capture` bare, `abcd launch` dry-run).
Bob runs it any time to see "what would the next release look like" without
touching the tree. **The durable prose is committed once at ship, not
regenerated per read** (see the determinism decision).

### 7. The plugin-distribution slice (itd-67)

The manifests already exist; the remaining work is small and mechanical:

- **`version` in `plugin.json`** — the sole in-file version, **rendered into the
  release payload only, never committed to the dev tree** (grilling outcome 4:
  ADR-19 keeps the working tree version-absent; the CHANGELOG dated heading is the
  one in-tree carrier). Kept in lockstep with `marketplace.json`'s
  `/plugins/0/changelog/version` (existing `lockstep.go` contract, release-mode).
  Seed is the **current newest CHANGELOG/tag, `0.3.0`** (grilling outcome 4 —
  corrects an earlier `0.1.0` error), so the first derived cut is
  next-after-`0.3.0` (`additive`→`0.3.1`, `breaking`→`0.4.0`).
- **README install/update path:** `/plugin marketplace add <personal account>/abcd` →
  `/plugin install abcd@abcd-marketplace`; update via `/plugin update abcd`.
- **Light installability smoke** (itd-67 AC): `marketplace.json` + `plugin.json`
  parse; `source` resolves; **every declared command/skill/agent/hook path
  exists** in the payload. A missing path FAILS. This is the light tier;
  itd-66's deep tier (import every Python entrypoint, render each command's
  help/frontmatter, isolated-subprocess) is the **later upgrade** this call is
  swapped to.

### 8. How ADR-37 is preserved and the auto-release grep stays valid

- ADR-37's instrument — _rolling `[Unreleased]` into a dated heading in a
  reviewed PR is the release decision_ — is **unchanged**. This programme only
  changes _who writes the roll_: an agent, not a human, and _who picks the
  number_: derivation, not a maintainer. ADR-37 §5 pre-authorised exactly this.
- The generation writes the **same heading shape** (`## [X.Y.Z] - <date>`) the
  grep already matches (`^## \[v?[0-9]+\.[0-9]+\.[0-9]+\] - `). **Zero workflow
  changes.** No superseding ADR is required; at most a one-line ADR-37 amendment
  noting itd-73 has landed (optional, grill-worthy).

---

## Resolved open questions (each a named decision + one-line why)

**RQ-1 — Determinism of LLM-generated prose in a durable record.**
_Decision:_ generate at ship and **COMMIT the generated prose into the ship PR**;
never silently regenerate. `abcd changelog` (bare) previews without committing.
_Why:_ a durable record must be reviewable and stable — reviewed once, then it is
history, not a per-read re-roll of a non-deterministic model.

**RQ-2 — Do resolved issues carry `impact`, or map from `severity`?**
_Decision:_ **intents are the primary version driver and get `impact`; resolved
issues default to `fix`** (contribute `Fixed` lines, patch-level), with an
**optional explicit `impact` override** for the rare issue that is actually
`breaking`/`additive`. _Why:_ issues are defect repairs by definition; forcing an
`impact` on every one adds ceremony for zero signal, while the override honestly
handles the edge (a "fix" that removes a surface is a `breaking` issue and must be
labellable). _Honest edge:_ an unlabelled breaking issue would under-bump — the
surface-diff guardrail (§4) is the backstop that catches the structural case.

**RQ-3 — "Shipped since last release" query.**
_Decision:_ a **git-tag-anchored set**, computed deterministically:
- resolve the last release tag (`vX.Y.Z` = newest dated CHANGELOG heading);
- **intents** = those whose move into `shipped/` landed **after** that tag
  (`git log <tag>..HEAD -- .abcd/development/intents/shipped/` → added paths);
- **issues** = those whose move into `resolved/` landed after the tag (same, over
  the issue ledger);
- **commits** = `git log <tag>..HEAD` for the prose agent's context.
_Why:_ the tag is the only immutable anchor; directory-as-truth + git history
gives a reproducible set with no new bookkeeping. Reuses `gitutil`.

**RQ-4 — Where the logic lives.**
_Decision:_ **deterministic** version derivation + surface snapshot/diff +
assembly = **Go**, in a new `internal/core/changelog` (version arithmetic reuses
`internal/core/launch/semver.go`; no separate `version` package unless the
arithmetic outgrows it), behind a Cobra `abcd changelog` / `abcd launch` front
door. The **prose generation is host-delegated (ADR-25)** to an agent, its output
validated and committed by the Go side. _Why:_ one-canonical-primitive — extend
`semver.go`, the lint host's folder scanner, `gitutil`, `fsutil`, and the launch
surfaces rather than reinventing; keep the non-deterministic part (prose) outside
the deterministic core.

**RQ-5 — Migration of the existing large `[Unreleased]` block.**
_Decision:_ the **first derived ship folds the existing hand-written
`[Unreleased]` body together with the newly-generated entries** into the first
dated section; **no retroactive per-record backfill.** _Why:_ the current
`[Unreleased]` lines are already good prose; discarding or re-deriving them is
wasted work and would lose detail no record carries. One clean fold, once.

**RQ-6 — `impact` on bundles.**
_Decision:_ **each member declares its own `impact`; the bundle's impact is the
max.** A newly-enforced discipline that breaks consumers counts as `breaking`.
_Why:_ mirrors the release-level aggregation (max impact wins) and matches
itd-73's own leaning; a bundle has no single product judgement of its own.

---

## Phasing — gated phases, each detector-first, its own PR, reviewed

Each phase declares a **STOP condition**: hitting it means stop and report, never
push through. Every phase is **detector-first** (write the failing test/lint,
watch it fail, then make it pass) per the repo's "wired or it isn't done" and
tests disciplines.

### Phase 0 — Plan the intents, confirm specs, record the deferral

- `abcd intent plan itd-73` (turn the draft into a planned, specced intent);
  confirm itd-67's spec covers the changelog + distribution slices as scoped
  here; **record the itd-66 deferral explicitly** (in itd-67's spec and a
  DECISIONS line).
- **Deliverable:** planned itd-73, confirmed itd-67 scope, dated deferral note.
- **STOP if:** planning itd-73 surfaces a scope conflict with itd-67/itd-66 that
  this plan has not resolved — stop and re-grill the boundary.

### Phase 1 — `impact` field + lint + version derivation (deterministic)

- Add `impact` frontmatter to the intent schema; add the `intent_impact_valid`
  blocker to `internal/core/lint`; back-fill existing shipped intents once.
- Implement `deriveNext` in `internal/core/changelog` over `launch/semver.go`,
  incl. the pre-1.0 table and the tag-anchored shipped-set query (RQ-3).
- **Detectors first:** lint test (missing/invalid impact blocks); table-driven
  derivation tests (each cell of §3, incl. pre-1.0 `breaking` → minor, empty-set
  → no bump).
- **Satisfies:** itd-73 AC 1–3, 5 (impact lint); the derivation half.
- **STOP if:** the shipped-since-tag query is not deterministic across a rebase
  or squash-merge — stop; the anchor model needs rework before proceeding.

### Phase 2 — Surface snapshot + guardrail

- Generate `surface.json` with a **new structured JSON emitter sharing
  `NewRootCommand()`** (outcome 5 — NOT `GenerateReference`'s Markdown walker,
  which excludes the hidden `hook` subtree and has no structured flag/manifest
  data); include per-flag required-ness + the manifest surface. **Seed the
  baseline** at the current release commit and add a **drift test** for
  `surface.json`; a cut with no baseline is fail-closed. Implement the ship-time
  diff + the "structural break needs a `breaking` intent" gate (taxonomy in
  outcome 5).
- **Detectors first:** a test that a removed command with no `breaking` intent
  FAILS the launch and names the surface; that a `breaking`-labelled removal
  passes.
- **Satisfies:** itd-73 AC 4 (surface-diff guardrail).
- **STOP if:** the surface walk is non-deterministic (ordering, transient
  hook subtree) — a flaky guardrail is worse than none; stabilise first.

### Phase 3 — Build `launch ship` + changelog generation + `abcd changelog` preview

- **FIRST, build the `abcd launch ship` write-verb + its orchestrating markdown
  command (outcome 10) — it does not exist today** (`launch` is dry-run-only). Two
  Go entry points mirroring `disembark`: an **emit** step (record-set + derived
  version + guardrail) and an **ingest** step (`--changelog-json` → bijection →
  write); the host runs the composer agent between them. Wire the
  `planned/`-closed-spec refuse gate (outcome 11) and the tag-ahead-of-heading
  in-flight refuse (outcome 1) here.
- **Author a new release-changelog composer agent** (outcome 9, named distinctly —
  NOT `CHANGELOG`, per iss-110) modelled on
  `press-release-composer`/`principle-distiller` (host-delegated, cite-or-be-dropped,
  injection-canary fixture, `prompt_version` per itd-5). Wire the ship-time flow
  (§5): assemble → derive → guard → **delegate prose to that agent (ADR-25),
  ingested via `--changelog-json`** → **run the completeness bijection (outcome 2)**
  → **clean-cutover: no fold; empty `[Unreleased]`, fully derived from this cut on**
  (outcome 7) → write the dated heading (+ render lockstep manifest versions into
  the release payload, outcome 4). Add the read-only, deterministic-only
  `abcd changelog` preview verb (outcome 6).
- **Detectors first:** a golden-ish test that assembly + validation produce a
  well-formed dated section from a fixture record set (validate structure, not
  exact prose — prose is the agent's); a test that `abcd changelog` writes
  nothing; a test that the written heading matches the `auto-release.yml` grep.
- **Satisfies:** itd-67's changelog-generation slice; ADR-37 preservation
  (heading-shape contract); iss-24 (no more `[Unreleased]` conflicts — per-cut
  generation replaces the shared hand-edited block).
- **STOP if:** the validated agent output cannot be made deterministic-enough to
  review (e.g. the agent is unreachable in the ship context) — stop; loud-stage
  and decide the CI/host boundary before landing.

### Phase 4 — Plugin manifest version + install path + light smoke

- Add `version` to `plugin.json` (lockstep with `marketplace.json`); document the
  install/update path in the README; implement the **light** installability smoke
  (parse + `source` resolves + declared surface paths exist).
- **Detectors first:** smoke test that a missing declared command/skill/agent/hook
  path FAILS; that a well-formed manifest passes.
- **Satisfies:** itd-67 distribution AC (marketplace resolves, install registers,
  version bumps in lockstep, smoke fails on a missing path).
- **STOP if:** the light smoke and itd-66's future deep smoke would diverge on
  what "the surface" is — align the surface-resolution seam now so the deep tier
  is a drop-in upgrade, not a rewrite.

---

## Scope boundary — what is explicitly NOT here

- **itd-66 (deep render + parity-diff + deep-smoke) is DEFERRED** to a follow-up.
  itd-67's own AC frames its smoke as _"light… later upgraded to call"_ itd-66's
  deep version, so Phase 4 builds the light tier with the surface-resolution seam
  positioned for itd-66 to slot in. **Ordering: this programme → itd-66.**
  itd-66 owns: materialised payload render, the `.abcd/**` leak-proof assertion,
  symlink resolution, parity diff vs the previous release, and the isolated-
  subprocess deep smoke (import every Python entrypoint, render each command's
  help/frontmatter). None of that is in scope here.
- **No pre-release channels** (alpha/beta/rc) — itd-73 out-of-scope; a later
  refinement.
- **No per-consumer compatibility ranges / deprecation windows** — downstream.
- **No behavioural-break detection** behind an unchanged surface — the author's
  `breaking` judgement, not the structural guardrail.
- **No superseding ADR for ADR-37** — it is preserved; at most a one-line
  amendment noting itd-73 landed.
- **No workflow changes** — `auto-release.yml` / `release.yml` are untouched.

---

## Risks / STOP conditions

- **Non-deterministic shipped-set across squash/rebase merges** (Phase 1 STOP).
  If a squash-merge collapses the `shipped/` move so `git log <tag>..HEAD --
  <path>` mis-reports, the tag-anchored query is unsound. _Mitigation:_ anchor on
  added paths in the range, test against squash and merge-commit histories.
- **Agent unavailable at ship in a CI-only context** (Phase 3 STOP). Prose is
  host-delegated (ADR-25); CI has no model. _Mitigation:_ the cut happens
  **host-side in the ship PR** (where the agent runs), then `auto-release.yml`
  only greps the committed heading — CI never needs the agent. Loud-stage if a
  cut is ever attempted where no agent is reachable; write nothing partial.
- **Guardrail false positives** (Phase 2 STOP). A flaky surface walk that flags
  a non-change blocks legitimate ships. _Mitigation:_ deterministic ordering;
  reuse the proven `GenerateReference` walker; test stability.
- **Under-bump from an unlabelled breaking issue** (RQ-2 edge). _Mitigation:_ the
  surface-diff guardrail catches the structural case; the explicit `impact`
  override on issues handles the rest; documented honestly, not hidden.
- **Migration fold loses detail** (RQ-5). _Mitigation:_ fold, do not
  re-derive, the existing `[Unreleased]` prose; review the first dated section
  in its PR.

---

## SOTA note (per `sota-per-intent`)

The comparable tools are the record-fragment / derived-changelog family:

- **towncrier** — per-change news fragments in a directory, assembled at release;
  iss-24 explicitly names this pattern ("concurrent PRs never conflict on the
  Unreleased block"). We adopt its **anti-conflict insight** (per-record source of
  truth) but our fragments _are the intents/issues themselves_, not a separate
  `newsfragments/` dir — no duplication.
- **changesets** (JS) — per-PR changeset files declaring a semver bump
  (major/minor/patch) + a summary, aggregated at release to derive the version.
  This is the **closest analogue** to `impact` + derivation; we place the bump
  declaration on the intent (`impact`) rather than a loose file, so it lives with
  the product judgement.
- **release-please / semantic-release** — derive the version from Conventional
  Commit prefixes and auto-open a release PR / tag. We deliberately derive from
  **records (intents' `impact`), not commit-message parsing** — commits are noisy
  and post-hoc, while `impact` is a deliberate product judgement set when the
  intent is shaped.

**Deliberate divergence, no new dependency.** The derive-from-records +
**host-generated prose** approach is chosen over adopting any of these tools:
they either parse commit messages (wrong source of truth for abcd) or template
fixed prose from fragments (abcd's locked decision is _agent-written_ prose from
the press-release/AC, richer than a fragment line). All logic reuses in-tree Go
primitives (`semver.go`, the lint host, `gitutil`, `fsutil`) + a host-delegated
agent — **no new module dependency**, matching `script-first-mvp` and
`host-delegated-by-default`.

---

## Traceability — which AC each phase satisfies

| Phase | itd-73 AC | itd-67 AC |
| --- | --- | --- |
| 1 — impact + lint + derivation | additive→minor, breaking→major, fix→patch (AC 1–3); impact lint blocker (AC 5); working-tree-only version (AC 6) | version lives in manifest + tags, not doc bodies |
| 2 — surface guardrail | surface break w/o `breaking` intent FAILS, names surface (AC 4) | — |
| 3 — changelog generation + preview | (feeds the derived number into the cut) | auto-recorded changelog generated from shipped intents + resolved issues; single `CHANGELOG.md` home |
| 4 — manifest version + install + light smoke | derived number lands on the release artefact (AC 6) | marketplace resolves; install registers `/abcd:*`; version bumps in lockstep; light smoke fails on a missing declared path |

**Deferred to itd-66 (follow-up):** deep render, `.abcd/**` leak-proof assertion,
symlink resolution, parity diff, isolated-subprocess deep smoke.

---

## Decisions to record (DECISIONS.md / ADR candidates)

- The bot-on-main alternative is rejected — it would reverse ADR-37; the reviewed
  ship stays the release decision (one line, DECISIONS.md).
- Changelog prose is host-delegated (ADR-25) and committed once at ship, never
  per-read regenerated (RQ-1) — candidate for a short ADR if it shapes the core
  boundary.
- Resolved issues default `impact: fix` with an optional override (RQ-2).
- itd-66 deferred as a follow-up; ordering this programme → itd-66 (Phase 0).
