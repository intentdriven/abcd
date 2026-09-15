---
id: itd-67
slug: installable-versioned-plugin
spec_id: spc-11
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-28]
prd_path: null
grill_session_id: 67d0f1de-0067-4a67-9c0d-000000000067
grilled_at: 2026-07-01
grilled_intent_hash: e549db8268a6bb1afee6c9a5a54a2703e9618957916cc485e390499718d2f053
glossary_terms_used:
- distribution/version
- distribution/release
- distribution/end-user
- core/brief
- core/intent
- core/phase
- core/spec
- interview/session
warrants_assumed:
- "The single abcd repo IS the marketplace; packaging excludes .abcd/** from the release artifact (adr-28)."
- "plugin.json.version as the sole in-file version is compatible with the doc-fidelity gate (a machine manifest field is not prose)."
builds_on: [itd-66]
severity: critical
---

# abcd Is An Installable, Versioned Claude Code Plugin Whose Repo Is Its Own Marketplace And Whose Every Launch Bumps, Tags, And Publishes A New Version

## Press Release

> **The abcd repo becomes a real Claude Code marketplace-and-plugin: it carries `.claude-plugin/marketplace.json` (source `./`) and a semver-stamped `plugin.json`, so an end-user runs `/plugin marketplace add REPPL/abcd` then `/plugin install abcd@abcd-marketplace` and gets the full `/abcd:*` command surface. Every `/abcd:launch ship` auto-selects a version bump (patch/minor/major per the brief's bump-tier rule), writes it into `plugin.json` and the marketplace entry, and tags the repo — so end-users update with `/plugin update abcd` and always pull a coherent, versioned release. The development record under `.abcd/**` is excluded from the release artifact by packaging ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)) — one repo is both the marketplace and the workshop.** Today abcd is developed but never CONSUMED as a plugin: there is no `.claude-plugin/` marketplace manifest, `plugin.json` carries no version, and there is no install or update path. This intent makes abcd actually installable and keeps it current — the precondition for anyone (including its own maintainers) to run the `/abcd:*` surface.

> "I've been building abcd but I can't actually install it — the repo has no plugin manifest and no version," said Kira, the maintainer. "Before we gate a launch payload, abcd has to BE an installable, updatable plugin. Adding the marketplace should be one command, updating should be one command, and every release should carry a real version number that falls out of the work we shipped."

## Why This Matters

Every other launch concern is downstream of this one. The pre-flight gate suite ([[itd-65-launch-preflight-gate-suite]]) and the payload render/parity/smoke ([[itd-66-launch-payload-render-parity]]) both assume there is an installable, versioned plugin to gate and render — but there isn't: the repo has no `.claude-plugin/`, so it cannot be added as a marketplace or installed, and `plugin.json` has no `version`, so Claude Code has nothing to compare on update. The canonical launch brief (`04-surfaces/04-launch.md` §§ 2, 4) already specifies that the curated release is the versioned artefact — a single repo whose packaging excludes `.abcd/**` (adr-28) — and that `launch ship` bumps `plugin.json`, updates `marketplace.json`, and refreshes version references — but nothing builds it, so the version story is design-only. This intent makes the distribution real and self-updating: it is the difference between "abcd is a repo you read" and "abcd is a plugin you install and keep current." It also closes a self-consistency gap the maintainer just hit — being unable to invoke `/abcd:intent` because abcd isn't installed as a plugin in a working session.

## What's In Scope

- The abcd repo carries `.claude-plugin/marketplace.json` (marketplace `abcd-marketplace`, one plugin `abcd`, `source: "./"`) AND the plugin's `.claude-plugin/plugin.json` with a real semver `version` — the single repo is simultaneously the marketplace and the plugin, with `.abcd/**` excluded from the release artifact by packaging (adr-28).
- A version field lands in `plugin.json` — the single source of the installed version; git tags on the repo are the canonical release points (per project standards, the version lives in the manifest + tags, not scattered across files).
- `/abcd:launch ship` auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection, writes the new version into `plugin.json`, updates `marketplace.json` (version + changelog entry), tags the repo, and records the tier + reason in the launch report.
- A dedicated, **auto-recorded changelog** in the repo, generated from the canonical history (git tags + commit/spec history + launch report) — never hand-curated prose. This changelog is the single home for "what changed and why it changed" (per the `abcd-cli/CLAUDE.md` docs-describe-present rule), and is the **reroute target** for [[itd-65-launch-preflight-gate-suite]]'s doc-history gate: change-narration that gate strips from a doc body is appended here.
- The install path is documented in the repo README (`/plugin marketplace add REPPL/abcd` → `/plugin install abcd@abcd-marketplace`) and the update path (`/plugin update abcd`).
- A smoke check that the published manifest is installable: `marketplace.json` + `plugin.json` parse, `source` resolves, and every declared command/skill/agent/hook path exists in the payload (shares the installed-surface assertion with [[itd-66-launch-payload-render-parity]]).

## What's Out of Scope

- The pre-flight security/PII/marker gate suite ([[itd-65-launch-preflight-gate-suite]]) and the payload render/parity mechanics ([[itd-66-launch-payload-render-parity]]) — this intent is DISTRIBUTION + VERSIONING, and reuses their manifest/smoke assertions rather than reimplementing them.
- A separate dedicated marketplace repo — decision is the single abcd repo IS the marketplace (source `./`, adr-28); multi-plugin marketplaces are a later concern.
- Manual per-ship version entry as the primary path — auto bump-tier is the default (per brief §4); `--version <x.y.z>` remains the override / major-bump escape hatch, not the norm.
- Auto-publishing to any registry beyond the git repo + tag — distribution is git-native marketplace, not a package index.
- Touching the wrapped dependencies' own versions — this versions abcd's OWN release, never the tools it wraps (wrap-only rule; distinct from the `dep_watcher` upstream-tracking machinery).

## Scope Conditions

None stated.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** the published abcd repo, **when** a user runs `/plugin marketplace add REPPL/abcd`, **then** the marketplace resolves and lists the `abcd` plugin from `marketplace.json` (source `./`).
- **Given** the added marketplace, **when** a user runs `/plugin install abcd@abcd-marketplace`, **then** the plugin installs and the full `/abcd:*` command/skill/agent/hook surface registers in a session.
- **Given** a shipped abcd with a recorded version, **when** a new `launch ship` publishes, **then** `plugin.json.version` is bumped by the auto-selected tier (patch/minor/major per brief §4), `marketplace.json` is updated, the repo is tagged, and `/plugin update abcd` pulls the new version.
- **Given** the bump-tier detection, **when** a phase completed since the last launch, **then** the bump is minor and the launch report names the completed phase; when none did, the bump is patch; a major bump occurs only via explicit `--version <x.0.0>`.
- **Given** the published manifest, **when** the installability smoke check runs, **then** `marketplace.json` + `plugin.json` parse, `source` resolves, and every declared command/skill/agent/hook path exists — a missing path FAILS the check.
- **Given** project standards forbidding git-inferable metadata in files, **when** the version is recorded, **then** it lives in `plugin.json` + git tags only, not duplicated across doc bodies.

## Open Questions

- Does `plugin.json.version` seed at `v0.1.0` (pre-1.0 signalling in-development) or does the first public install imply `v1.0.0`? The bump-tier rule needs a defined starting point.
- Where does the changelog live — a `CHANGELOG.md` in the repo, the `marketplace.json` entry, or git tag annotations — and is it generated from the launch report or hand-curated?
- How does auto bump-tier detection read "phase completed since last launch" before the `phase:` frontmatter anchor is active (brief §4 notes it falls back to editorial `## Scope` membership until then)?
- Should `launch ship` refuse to publish if `plugin.json` version would not change (nothing new since last tag), or always allow a forced patch re-snapshot?
- Does the install-path documentation belong only in the repo README, or also mirrored in `docs/` for the plugin's own help surface?
