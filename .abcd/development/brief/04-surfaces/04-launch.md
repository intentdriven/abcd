# `/abcd:launch` — Curated Release

> **Phase ownership** ([adr-33](../../decisions/adrs/0033-launch-phase-ownership-tiered.md)): the curated-release cut — packaging with `.abcd/**` excluded plus the secret/PII scan — ships in [Phase 1](../../roadmap/phases/phase-1-ahoy.md). The full pre-flight gate suite and remaining release automation below are separately scheduled intents (itd-65 gate suite, itd-66 render parity, itd-70 retention, itd-72 tier-b publishing); itd-73 derived versioning ships with the release cut (see [Sub-verbs](#sub-verbs)).

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`), verified
> against the committed command-tree snapshot in both directions._

| Verb | Bucket | Status |
|---|---|---|
| `scaffold` | — | shipped |
| `ship` | gate | shipped |


The shipped verb surface is the `--dry-run` flag on `abcd launch` — a read-only preview of the bundle and gates — plus the `abcd launch ship` subcommand, which cuts the release (the RELEASE CUT slice only; see the ship bullet below), and the `abcd launch scaffold` subcommand, which writes the release machinery into a managed repo (itd-93; see the scaffold bullet below). Bare `abcd launch` never mutates state: it refuses (exit 1) with a hint to pass `--dry-run`; a bare-as-status render is a design target, unshipped. The sub-verb design:

- **`/abcd:launch ship`** — **partly shipped: the RELEASE CUT only** (itd-73 derived versioning + itd-67's changelog slice). `abcd launch ship` derives the version and the record set from what shipped since the newest tag, runs the surface guardrail, and — with `--changelog-json` — validates the host-composed prose against the record set (the completeness bijection), admits only the `Added` and `Fixed` sections — the composer sees the records that shipped and never the previous release's surface, so `Changed`, `Deprecated`, `Removed` and `Security` are refused by name and each dated section states under its heading what the notes list and do not claim (iss-2609011207114761) — and writes the dated `CHANGELOG.md` heading `.github/workflows/auto-release.yml` turns into a tag; `commands/launch.md` carries the emit → compose → ingest orchestration and the `release-changelog-composer` agent it dispatches. The `Ship` engine is wired: `abcd launch ship` is a live subcommand, and `--payload-dir <dir>` stages the versioned release payload — the derived version stamped into the payload's `plugin.json`/`marketplace.json` and lockstep-proved before return. Commit, tag, and publish stay a design target (itd-65 gate suite + itd-72 publishing): the verb neither commits, tags, nor publishes. The full-cut design: cut a curated release artefact from the one repo: run pre-flight gates, filter the artefact (default-deny, `.abcd/**` excluded by packaging), stamp the version, and on a `v*` tag publish a GitHub Release ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)). The flow described in §§ 1–6 below is this sub-verb's behaviour. Flag-shaped modifiers `--allow-dirty` and `--allow-doc-warnings` belong to this sub-verb's design; the shipped `ship` accepts `--changelog-json` and `--payload-dir` (plus the global `--json`), and bare `abcd launch` accepts only `--dry-run` and the global `--json`. There is no version flag — the version is derived, never authored ([adr-31](../../decisions/adrs/0031-derived-versioning-from-intents.md), see [§ 3](#3-versioning--marketplace)).
- **`/abcd:launch scaffold`** — **shipped** (itd-93, spc-14). `abcd launch scaffold` writes the changelog-driven release machinery — `.github/workflows/release.yml`, `.github/workflows/auto-release.yml`, and the adr-37 release runbook (`.abcd/development/release-gate/README.md`) — into a managed repo that lacks it, wired to the repo's own default branch and Go version, `GITHUB_TOKEN`-only and injection-safe. The workflows ship from a single embedded template that abcd-cli's own release workflows are regenerated from (self-scaffold parity, a byte-exact test); the scaffolded `release.yml` carries a `workflow_dispatch` **rehearsal** that arms the full gate against a simulated changelog roll and reviewed-content commit and publishes nothing, so a green rehearsal is the runbook precondition for the first real release. A bare repo with no semantic detector degrades cleanly to the deterministic gates and a generic build. It is idempotent and fail-safe: a re-run on current machinery is a no-op (exit 0), a hand-edited file is refused (exit 1) rather than clobbered unless `--confirm` is passed, and a structural fault exits 2. `commands/launch.md` carries the flow. Accepts `--confirm` plus the global `--json`.
- **`/abcd:launch dry-run`** — shipped as the `--dry-run` flag (the plugin command `commands/launch.md` maps the `dry-run` of its `[dry-run | ship | scaffold]` argument hint onto `abcd launch --dry-run`; the binary has no `dry-run` subcommand). **Report-only preview, always exit-0** (a preview never blocks). It runs the parts of the pre-flight suite that exist today: as of spc-64 (predecessor store) the **secret + PII scan gate** (the native scanners, see [§ 1](#1-pre-flight-gates)) runs for real in report-only mode and prints what it *would* refuse on (a finding, or a fail-closed reason such as "scanner unavailable"); the **installability smoke** runs for real at its light tier (see [§ 1](#1-pre-flight-gates)); the **manifest lockstep check** runs for real at its `dev` polarity over the working tree — the polarity adr-19 requires the committed manifests to satisfy (no version key), see [§ 3](#3-versioning--marketplace) — reports its result and folds any drift or an unreadable version-location contract into what it *would* refuse on; the **citation-baseline gate** (`internal/core/launch/citations.go`) runs for real, tallying the cited claims against their receipts (e.g. `51 cited, 51 with receipts`); the **semantic-receipts row** (`internal/core/launch/receipts.go`) reports, as `{"status": "host-run"}`, which semantic-pass receipts are recorded for the candidate commit — presence only, never a verdict, because `release.yml` owns the required-gates list and judges receipt validity (iss-2608231226342272; the row exists because a preview silent about `receipt_gate` let a one-commit release branch reach a tag and fail-close there); the remaining gates (marker-block, documentation-auditor) are the gate-suite intent's (itd-65): `--dry-run --json` reports each as `{"status": "not_implemented", "detail": "Phase-5 deferred"}`, while the plain-text `--dry-run` render omits the gate list entirely (it prints version, files bundled, scan hardfails, citations, receipts, and would-publish, plus a would-refuse-on line when there is a finding). It also produces the would-be artefact manifest, without writing the release artefact. dry-run is **not** "ship minus publish": running the *full* gate suite and **hard-failing** on a finding (exit non-zero) is the full `ship` verb's behaviour (itd-65 + itd-72), not dry-run's.

## 1. Pre-flight gates

- **Secret scan** — a **native Go scanner** is the default, hard-fail (absent/fail-closed, never a silent skip). **gitleaks** is an opt-in deeper scanner (the spc-64 (predecessor store) ship gate pins `gitleaks >= 8.18.0` when wired; absent/older = fail-closed, never a regex fallback).
- **PII** scan (real names, emails) via the **native Go PII engine** (`scan_text` + the merged Config + the non-overridable secret/identity severity floor) — hard-fail.
- **Custom regex** layer — home dirs (`~/...`) and local usernames — hard-fail; GitHub usernames from git config — warn (legitimate in repo-URL contexts). The per-kind severity floor is config-raisable, never lowerable (`internal/adapter/scanner/identity.go`).
- **TruffleHog** — opt-in deep scan when `scan.deep=true` — hard-fail (live credential verification)
- **Hook compliance** check — warn-fail
- **Marker block sanity** — hard-fail on malformed
- **Installability smoke** (`plugin.json` parse + `marketplace.json` references) — hard-fail. Two tiers over ONE surface resolution (`internal/core/launch/installsurface.go`), so the deeper tier upgrades the assertions and never redefines the surface:
  - **The declared surface** is the UNION of two registers, because the plugin-manifest schema defines every explicit key as declaring entries *in addition to* the convention directory: **convention** — `commands/**/*.md`, `agents/*.md` (a flat glob; iss-110 is the evidence), `skills/*/SKILL.md`, `hooks/hooks.json` — and **manifest** — the optional `commands`/`agents`/`skills`/`hooks` keys in `plugin.json`. A hook's `$CLAUDE_PLUGIN_ROOT`-rooted command adds the payload file it invokes; the plugin executable itself is install-supplied, so its absence from the payload proves nothing. Every entry records which register declared it. This is NOT the compatibility surface (`internal/core/surface`), which records manifest KEYS and discards values; installability is the mirror question, over the values.
  - **Light tier** (itd-67, shipped) — both manifests parse, each local marketplace `source` resolves to a manifest whose name matches the listing, and every declared path the payload is responsible for is carried. Resolution reads the **resolved bundle**, so a file present in the tree but excluded from the payload fails here. `dry-run` reports it; `RenderPayload` refuses on it, because the render is the only step that materialises an artefact.
  - **Deep tier** (itd-66, deferred) — over the same resolved list: import every shipped entrypoint and render each command's help/frontmatter in an isolated subprocess rooted at the rendered payload.
- **Citation baseline** (`internal/core/launch/citations.go`) — shipped; runs
  for real in `dry-run`, verifying the committed citation baseline (every cited
  claim carrying a receipt) and reporting its tally.
- **Dirty tree** — refuse unless `--allow-dirty`
- **OWASP / vulnerability check** (folded into the pre-flight suite) — warn-fail
- **Documentation auditor** (subagent) — runs over `docs/` to verify user-facing documentation is well-formed before release — warn-fail

Pre-flight report written to `.abcd/logbook/launch/<timestamp>/preflight.{json,md}` — **full-`ship` behaviour (itd-65)**. The spc-64 (predecessor store) secret/PII gate is itself side-effect-free w.r.t. the repo (its only writes are to a private temp tree it removes), and `dry-run` renders the gate result inline rather than writing a report file.

## 2. Curated release artefact (default-deny)

- **Include:** the shipped include list, pinned in `.abcd/config/launch-payload.json`: `.claude-plugin/` (holds both `plugin.json` and the ONE canonical `marketplace.json` — there is no root-level `marketplace.json`), `commands/`, `agents/` (the reviewer/synthesis prompt catalog), `hooks/` (`hooks.json` is load-bearing), `scripts/`, `docs/` (user-facing only), `README.md`, `LICENSE`, `.gitignore`. `skills/` is absent from the include list and does not exist in the tree at all.
- **Exclude:** `.abcd/` (entire namespace — `development/` (brief, decisions, intents, plans, principles, research, roadmap), plus the design-target tiers `memory/` and `logbook/` once they exist), and patterns from `.gitignore`. Per [adr-28](../../decisions/adrs/0028-single-repo-curated-release.md) the wholesale `.abcd/` exclusion (incl. `.abcd/memory/**`) is a **packaging filter over the one tree**, not a copy between two repos: the release artefact carries plugin code, never the project's design record or knowledge store. The lifeboat is **not** among the excluded tiers because it is not an in-tree tier at all: `/abcd:disembark` is read-only over the source repo and lands the lifeboat at an operator-chosen destination outside it, with its voyage log at the operator level under `~/.abcd/voyage/` ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)) — so there is nothing for the payload filter to exclude. The spc-38 (predecessor store) restrictive-licence gate is NOT this artefact's gate — its real consumer is the lifeboat (`/abcd:disembark`), the surface that publishes curated project memory/provenance ([adr-35](../../decisions/adrs/0035-lifeboat-as-coverage-experiment.md)). At launch the gate is future/inert; the shipped `dry-run` renders no licence verdicts (a diagnostic preview of the gate's verdicts in `dry-run` is part of the gate's own design, unshipped).
- **Override:** the include list in `.abcd/config/launch-payload.json` is the packaging override (the only mechanism that can put a path *into* the release artefact). The deny is **structural** (`internal/core/launch/bundle.go`, per adr-28): no include entry can promote a denied namespace — a `.abcd/**` line is never promoted — so nothing can re-include `.abcd/memory/**`. The spc-38 (predecessor store) gate's own evaluation-input allowlist belongs to that gate's design (future/inert, see above) and is documented-distinct from packaging: it re-includes files into the gate's *own evaluation input*, never into the release artefact — two mechanisms, never one name.

## 3. Versioning + marketplace

The curated release artefact is the only abcd artefact that carries a semantic
version. Versions are an *output* of cutting a release, never a sequencing input
on the design record — the repo organises work by **phase** (see
[adr-9](../../decisions/adrs/0009-phase-as-product-layer.md)), and a release
number is what falls out when a stretch of that work is published. The brief,
intents, and roadmap carry no version label.

Versioning is **strict SemVer**: the version string is `MAJOR.MINOR.PATCH` (no
leading `v`) at the selected version location, and the release's git tag is
`v<version>`. While the version is `0.y.z` the operator surface may still change
between minor versions (pre-1.0 = not yet surface-stable); `1.0.0` marks the
first stable `/abcd:*` surface. The tag drives identification: what is installed
vs what is available is compared through the tagged, released artefact — the
working tree carries no version at all (adr-19, adr-28).

### Bump-tier rule

The version is **derived, never authored**
([adr-31](../../decisions/adrs/0031-derived-versioning-from-intents.md)):
`launch ship` selects the bump tier from the intents shipped since the previous
release; the tier and its reason are recorded in the launch report and the
commit message so every published version is traceable to *why* it bumped.

| Tier | Trigger |
|---|---|
| **Major** (`vx.0.0`) | Any shipped intent since the last release carries `impact: breaking`. |
| **Minor** (`v0.x.0`) | No `breaking`, at least one `impact: additive`. |
| **Patch** (`v0.0.x`) | Only `impact: fix` intents (or a release with no intent-tied change). |

**Impact derivation.** Every intent carries `impact: additive | breaking | fix`,
set when the intent is shaped and enforced by `internal/core/lint` (adr-31, tracked
by itd-73). At release, `launch ship` gathers the intents shipped since the
previous release and takes the highest-severity impact. A change not tied to any
intent falls back to conventional-commit derivation (`feat:` / `fix:` /
`feat!:` prefixes since the last tag).

**Surface-diff guardrail.** `launch ship` snapshots the `/abcd:*` command, flag,
and manifest surface and compares it to the previous release. A removed or
altered surface with no `breaking` intent in the release **fails the launch** —
a mislabelled impact cannot ship a compatibility lie.

**Unfixed-findings guardrail** (`internal/core/changelog.GuardFindings`, wired at
`internal/core/release/emit.go`). `launch ship` and the read-only `changelog`
preview both ask one further question of the cut: of the findings THIS CYCLE
produced, is any of them consequential, still open, and unanswered? A cut that
carries one is REFUSED, under the refusal kind `unfixed-finding`, and a refused
cut carries no derived version at all. The gate fired on this release's own
first cut, so it is live behaviour rather than a design target.

- **What counts as this cycle's** is a set difference of issue-ledger membership
  between the anchor tag's tree and `HEAD`, keyed on the record id across all
  three status directories. The id survives a re-slug and a move to a terminal
  folder, which a path does not, so a standing backlog record that merely moved
  is never reported as newly captured. The standing backlog is out of scope by
  construction: a gate that blocked on it would refuse every release until the
  whole ledger was drained, which is how a gate gets switched off rather than
  satisfied.
- **What blocks** is a severity of `major` or `critical`, and also a severity
  that is absent, misspelled, or outside the ledger's enum. An unreadable grade
  has not been judged, and "not judged" must not read as "not serious".
- **The waiver** is the frontmatter pair `deferred_after` + `deferral_reason`,
  both schema-accepted keys (`internal/core/issueschema`). `deferred_after` names
  the cut's ANCHOR tag, not the version being derived, which is what makes a
  waiver single-use: the anchor moves at the next release and every waiver
  written against the old one lapses, so a deferred finding is re-asked rather
  than forgotten. Half a waiver does not stand: one field without the other, or
  an anchor that is not this cut's, leaves the record blocking and reports why.
- **What the render shows.** `renderCut` prints a `findings:` line on every
  render of `abcd changelog` and `abcd launch ship` — the verdict, the count of
  unfixed findings and the anchor they were measured from, and the count
  deferred — then one `deferred:` line per waiver naming the record, its
  severity and its stated reason. A deferral an operator cannot see in the
  report they actually read is indistinguishable from having ignored the
  finding, which is the thing the waiver exists to be the opposite of. The whole
  verdict is on the `findings` key of the cut's JSON.
- **The four routes out**, in the order the refusal itself states them: fix the
  defect and resolve its record inside the cut; record the decision not to fix
  it (`abcd capture wontfix`, which clears the gate with no special case,
  because a wontfix carries a reason and is the cited non-action the rule asks
  for); defer it out loud with the waiver pair; or re-grade the record honestly
  when the severity was wrong in the first place. The gate asks only whether the
  record is still in `open/` at `HEAD`, so deleting it outright clears the gate
  as well — that is a hole, not a fifth route, and the honest answers are the
  four above.

`launch ship` is responsible for writing the version into **the selected
version location**, never a hard-coded `plugin.json`. That location is read from
the spc-77.1 (predecessor store) decision artifact
(`.abcd/config/version-location.json`)
as `manifest_path` + `json_pointer` (see
[adr-19](../../decisions/adrs/0019-plugin-json-version-carve-out.md)). The
artifact records the predecessor's spc-77.1 ACCEPT outcome — `.claude-plugin/plugin.json` at
`/version` — and the shipped lockstep checker fails closed when it is missing or
malformed; a `blocked: true` decision has no schema-valid location, so
version-writing refuses and the escalation stands. Concretely, `ship`:

1. Stamps the bumped version into the **release artefact** at the selected
   `manifest_path` + `json_pointer` — the manifest renderer reads the decision
   artifact and never parses a location string.
2. Leaves the **working-tree** manifests UNVERSIONED. Per
   [adr-19](../../decisions/adrs/0019-plugin-json-version-carve-out.md) and
   [adr-28](../../decisions/adrs/0028-single-repo-curated-release.md) the version
   is single-sourced in the *cut artefact*; the repo's committed manifests carry
   no version, so there is nothing to keep in sync on the working-tree side. The
   renderer stamps the version into the artefact content only — it never mutates
   the working-tree manifests.
3. Records the version + changelog entry in the marketplace metadata at the ONE
   canonical `.claude-plugin/marketplace.json` (never a root-level copy).
   **Later phase** ([adr-20](../../decisions/adrs/0020-manifest-version-lockstep.md)):
   the changelog entry conforms to a `changelog-entry.schema.json`, validated
   programmatically by this bump step.
4. Refreshes any other version references generated from the config slug.

**Anti-drift (present state).** The two manifests in the artefact describe one
release, so they must stay version-consistent: the version at the selected
location and the marketplace entry's version + changelog must agree. A read-only
lockstep checker proves this over the pinned path list
[adr-20](../../decisions/adrs/0020-manifest-version-lockstep.md) records; a
half-state (a version in one manifest and not the other) is drift. `ship`'s bump
step runs it against the staged release artefact at the **public** polarity
(primary present, every secondary agreeing) and refuses to publish on drift; the
**dev** polarity runs in `dry-run` over the working tree, asserting the committed
manifests carry no version key (adr-19), and folds any drift into the preview's
would-refuse-on set. The checker has no bypass flag, and adr-20 records that
`--allow-dirty` must not bypass manifest consistency (wiring policy, enforced by
the pre-flight suite).

Commit message — **full-`ship` design target (itd-65 + itd-72); the shipped
`ship` never commits, and no shipped path produces this format** (the release
cuts made so far use hand-written `chore:` prose): `chore(release): launch abcd
v<version> (<tier>: <reason>) from <source-sha>` — e.g. `chore(release): launch
abcd v0.3.0 (minor: additive itd-40 shipped) from a1b2c3d`.

### Release cut + retention

Every `launch ship` **cuts a release**: the release commit, the `v<version>`
git tag on it, the marketplace changelog entry, and — on the `v*` tag — a
published GitHub Release with **SLSA provenance** attached to the artefact
([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)) all describe
one released snapshot. The version lives only on the tag and in the cut artefact;
the working tree is never versioned (adr-19, adr-28).

Retention is **newest-per-line**: each release line (`MAJOR.MINOR`) keeps only
its newest release. Shipping `v0.1.2` removes the superseded `v0.1.1` — its git
tag and the GitHub Release and its assets — while shipping `v0.2.0` keeps the
last `v0.1.x` alongside it (the last release of every previous line survives as
that line's terminal snapshot). Three safety rules bound the prune:

- The release just published is **never** pruned.
- Pruning **refuses** if a release newer than the one just published already
  exists (out-of-order ship — resolve manually, never auto-delete forward).
- Retention prunes **release tags and Releases only**; git history is untouched.
  The launch report under `.abcd/logbook/launch/<timestamp>/` is the durable
  record of every launch, including pruned ones — deleting a release tag never
  deletes the evidence a launch happened.

A prune is a destructive, outward-visible act, so `ship` reports exactly which
release it removed (or why it refused) in the launch report and the ship
transcript; a `--dry-run`-shaped preview of the prune decision is part of the
`dry-run` artefact preview.

## 4. Reports

`launch-report.{json,md}` in the repo's `.abcd/logbook/launch/<timestamp>/` —
**full-`ship` behaviour (itd-65)**: no shipped path writes this layout yet, and
`.abcd/logbook/` does not exist in the tree.

## 5. Bootstrap exception

The first release cut of abcd itself is a manual `v*` tag + GitHub Release. Documenting the exception in `commands/launch.md` lands with publishing (itd-72); the shipped command file covers the read-only dry-run preview and the release cut (emit → compose → ingest), neither of which publishes.

## 6. Acceptance

- **Given** any abcd-aware terminal, **when** the user runs bare `/abcd:launch`, **then** the dispatcher shows current launch readiness (pre-flight gate state, last launch attempt timestamp), the available sub-verbs (`ship`, `dry-run`), and suggested next actions — bare invocation never mutates state. **Design target:** the shipped bare `abcd launch` refuses (exit 1) with a hint to pass `--dry-run`, and the shipped plugin command runs the dry-run preview directly rather than a status+help render.
- **Given** a clean tree with a deliberate PII fixture (e.g., a real email in a comment) inside the resolved artefact, **when** `/abcd:launch dry-run` runs, **then** the report-only gate (spc-64, predecessor store) PRINTS that it *would* refuse on that finding (the offending file/line in the gate result), still **exits 0**, and writes no artefact. (The **hard-fail** on that finding — exit non-zero plus a `preflight.{json,md}` report under `.abcd/logbook/launch/<timestamp>/` — is the full `ship` verb's behaviour (itd-65), not dry-run's.)
- **Given** a clean tree, **when** `/abcd:launch dry-run` runs, **then** the report lists exactly the include/exclude artefact manifest in [§ 2](#2-curated-release-artefact-default-deny) with no surprises and no artefact is written.
- **Given** only `impact: fix` intents (or no intent-tied change) shipped since the last release, **when** `launch ship` runs, **then** the bump tier is **patch** (`v0.0.x`) and the next patch version is written into the **selected version location** (from `.abcd/config/version-location.json`, per [§ 3](#3-versioning--marketplace)) in the **release artefact** only — the working-tree manifests stay unversioned (adr-19, adr-28) — plus the canonical `.claude-plugin/marketplace.json`, never a hard-coded `plugin.json`.
- **Given** at least one `impact: additive` intent and no `breaking` intent shipped since the last release, **when** `launch ship` runs, **then** the bump tier is **minor** (`v0.x.0`) and the launch report names the intents that drove it.
- **Given** any `impact: breaking` intent shipped since the last release, **when** `launch ship` runs, **then** the bump tier is **major** (`vx.0.0`) and the launch report names the breaking intent(s).
- **Given** a command, flag, or manifest surface removed or altered since the previous release with no `breaking` intent in the release, **when** `launch ship` runs, **then** the surface-diff guardrail **fails the launch** (adr-31) — the mislabel is reported, nothing is published.
- **Given** an issue record captured since the anchor tag, graded `major` or `critical` (or carrying no readable grade), still in `open/` and carrying no standing waiver, **when** `launch ship` or `abcd changelog` runs, **then** the cut is REFUSED under the `unfixed-finding` kind, the refusal names every such record with its grade and path, no version is derived, and the render's `findings:` line says how many were counted and from which anchor.
- **Given** the same record carrying `deferred_after` set to that cut's anchor tag and a non-empty `deferral_reason`, **when** the cut is emitted, **then** the gate PASSES and the render carries a `deferred:` line naming the record, its severity and its reason — and the same record blocks again at the next release, because the anchor has moved and the waiver has lapsed.
- **Given** any `launch ship` run, **when** the release commit is written, **then** the commit message records the bump tier and its reason (e.g. `(minor: additive itd-40 shipped)`).
- **Given** a documentation-auditor warn-fail, **when** `launch ship` runs without `--allow-doc-warnings`, **then** the user is shown the warnings and asked transparently whether to proceed.
- **Given** a prior release of the same line (`vX.Y.(Z-1)`) exists, **when** `launch ship` publishes `vX.Y.Z`, **then** the superseded release's tag and GitHub Release + assets are removed, the removal is named in the launch report, and the last release of every *other* line is untouched.
- **Given** a release newer than the just-published version already exists, **when** the retention step runs, **then** it refuses to prune anything and the launch report records the refusal reason.
