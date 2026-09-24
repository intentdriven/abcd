---
id: itd-67
shipped_in: v0.4.0
slug: installable-versioned-plugin
spec_id: spc-11
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-28, adr-2609231048308186]
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
impact: additive
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

- **Given** the published abcd repo, **when** a user runs `/plugin marketplace add REPPL/abcd`, **then** the marketplace resolves and lists the `abcd` plugin from `marketplace.json`, sourced from the latest release's pinned archive (`{"source": "archive", "url": ".../releases/download/vX.Y.Z/abcd-plugin-vX.Y.Z.zip", "sha256": "<digest>"}`).
  _Amended 2026-09-23 (source `./` → the pinned archive), on the product thinker's rulings E1 and E2 of that day: a relative-path source installs the unversioned working tree, so no install or update could receive a version-stamped release, which is what criterion 3 promises; the host harness reads a published release only through an archive source. The decision and the alternatives weighed are [adr-2609231048308186](../../decisions/adrs/2609231048308186-the-catalog-pins-the-latest-release-s-plugin-archive.md), which amends adr-19 and adr-20. Until the first release past v0.9.0 is cut, the committed catalog still carries `./` (that ADR's bootstrap)._
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

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-7af1556ce4f7 -->
Fidelity review — receipt rcp-7af1556ce4f7 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:5c2255f4a1fe91bd174e3569f9aa39f48753cc8121b7cc595d5948c949815ac0
Input attestations: diff:tree at de3ba5fa (spc-11 delivered; main after PR #661)@sha256:4e7430d38ef6b7b6bc533fdf0b8b32a6e08a3e2449d0566027d43316036b8178;

Acceptance rollup: MET 1 · MET_WITH_CONCERNS 3 · NOT_MET 2 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: marketplace.json names abcd-marketplace with one plugin abcd at source ./ and the light smoke resolves it over the committed payload; the criterion's REPPL/abcd slug is stale (README documents [redacted-user]/abcd, marketplace owner url still points at REPPL) and the host-side add cannot be exercised in-tree
  evidence: .claude-plugin/marketplace.json:3 — ""name": "abcd-marketplace""
  evidence: .claude-plugin/marketplace.json:11 — ""source": "./""
  evidence: internal/core/launch/smoke_test.go:13 — "func TestSmokeLightPassesOnCommittedPayload"
  evidence: README.md:83 — "/plugin marketplace add [redacted-user]/abcd"
  evidence: .claude-plugin/marketplace.json:6 — ""url": "https://github.com/REPPL""
- ac-2 — MET_WITH_CONCERNS: the light smoke asserts every declared command/agent/skill/hook path exists in the resolved bundle and passes over the committed payload; it asserts path presence only, not that the surface registers in a session (itd-66's deep tier is deferred by spc-11)
  evidence: internal/core/launch/smoke.go:110 — "for _, e := range surface.Entries {"
  evidence: internal/core/launch/smoke.go:118 — "declared %s %q is not in the payload"
  evidence: internal/core/launch/smoke_test.go:13 — "func TestSmokeLightPassesOnCommittedPayload"
  evidence: internal/core/launch/smoke.go:6 — "The light tier asserts the three things itd-67 names"
- ac-3 — NOT_MET: promised: a ship bumps plugin.json.version, updates marketplace.json, tags, and /plugin update pulls the new version; delivered: the version is stamped only into a payload staged outside the repo when --payload-dir is passed, the committed manifests carry no version at all, release.yml uploads binaries and checksums only, and the marketplace source ./ is the git tree — so /plugin update abcd never pulls a plugin.json.version; only the tag half arrives, via auto-release.yml
  evidence: internal/surface/cli/ship.go:224 — "if payloadDir != "" && ingested.Written {"
  evidence: internal/core/launch/render.go:365 — "func stampMarketplace(dest string, req PayloadRenderRequest) error {"
  evidence: .claude-plugin/plugin.json:3 — ""name": "abcd","
  evidence: internal/core/launch/render_test.go:94 — "func TestRenderPayloadLeavesSourceTreeUnversioned"
  evidence: .github/workflows/release.yml:327 — "gh release create "${TAG}" bin/abcd-* bin/checksums.txt"
  evidence: .github/workflows/auto-release.yml:3 — "tag-and-release the NEWEST dated"
- ac-4 — NOT_MET: promised: phase completion drives a minor bump named in the report, patch otherwise, major only via explicit --version; delivered: the tier derives from record impact (pre-1.0 additive is a patch, breaking a minor, 1.0.0 never derivable), the report names the deciding impact and record and no phase, and the ship verb carries no --version flag at all — spc-11 records the supersession but the criterion was never amended
  evidence: internal/core/changelog/version.go:38 — "func DeriveNext(prev launch.Semver, bump Impact) (launch.Semver, bool) {"
  evidence: internal/core/changelog/version.go:29 — "NO input can derive 1.0.0 from a 0.x base"
  evidence: internal/surface/cli/ship.go:143 — "return string(cut.Impact) + ": " + strings.Join(cut.DecidedBy, ", ")"
  evidence: internal/surface/cli/ship.go:262 — "cmd.Flags().StringVar(&changelogJSON, "changelog-json""
  evidence: .abcd/development/specs/closed/spc-11-installable-versioned-plugin.md:107 — "Bump-tier selection."
- ac-5 — MET: SmokeLight parses both manifests, resolves the marketplace source to a plugin manifest of the same name and asserts every declared path; a missing path is a finding that fails the smoke and blocks the ship, with one negative test case per failure kind
  evidence: internal/core/launch/smoke.go:56 — "func SmokeLight(tree PayloadTree) SmokeReport {"
  evidence: internal/core/launch/smoke.go:127 — "report.OK = len(report.Findings) == 0"
  evidence: internal/core/launch/smoke_test.go:26 — "func TestSmokeLightFailsAndNamesTheMissingPath"
  evidence: internal/core/launch/smoke_test.go:109 — "func TestRenderPayloadRefusesAnUninstallablePayload"
  evidence: internal/core/launch/ship.go:68 — "report.Smoke = SmokeLight(NewBundleTree(bundle))"
- ac-6 — MET_WITH_CONCERNS: no version is duplicated across doc bodies: the committed plugin.json carries none, the [redacted-user]-polarity lockstep test asserts the keys absent, and git tags are the release points; the concern is that the in-tree carrier is the CHANGELOG dated heading rather than plugin.json (adr-19/adr-37), so the version does not live in plugin.json + tags as the criterion says
  evidence: internal/core/launch/lockstep_repo_test.go:18 — "func TestCommittedTreeSatisfiesDevPolarity"
  evidence: internal/core/launch/lockstep.go:45 — "[redacted-user]: those keys must all be ABSENT"
  evidence: internal/surface/cli/ship.go:99 — "func publishedVersion(repoRoot string) string {"
  evidence: .github/workflows/auto-release.yml:81 — "tag="v$version""

Gap audit:
- honoured:
  - the repo carries marketplace.json (abcd-marketplace, plugin abcd, source ./) and plugin.json
    evidence: .claude-plugin/marketplace.json:11 — ""source": "./""
    evidence: .claude-plugin/plugin.json:3 — ""name": "abcd","
  - a light installability smoke that fails on a missing declared path
    evidence: internal/core/launch/smoke.go:56 — "func SmokeLight"
    evidence: internal/core/launch/smoke_test.go:26 — "TestSmokeLightFailsAndNamesTheMissingPath"
  - install and update path documented in the README
    evidence: README.md:89 — "/plugin install abcd@abcd-marketplace"
    evidence: README.md:98 — "/plugin update abcd"
  - an auto-recorded changelog derived from records, never hand-curated
    evidence: internal/surface/cli/ship.go:176 — "derive the version and the record set from what shipped"
    evidence: internal/surface/cli/ship.go:284 — "Preview the next release cut"
  - the version is not scattered across files
    evidence: internal/core/launch/lockstep_repo_test.go:18 — "TestCommittedTreeSatisfiesDevPolarity"
- diverged:
  - bump tier auto-selected per brief phase completion — delivered from record impact, with a pre-1.0 row that maps additive to patch
    evidence: internal/core/changelog/version.go:21 — "prev is 0.x breaking -> minor++"
  - launch ship tags the repo — delivered as auto-release.yml tagging the newest CHANGELOG heading on merge
    evidence: .github/workflows/auto-release.yml:3 — "tag-and-release the NEWEST dated"
  - the version lives in plugin.json — delivered as the CHANGELOG dated heading in-tree and a stamped payload manifest out-of-tree
    evidence: internal/surface/cli/ship.go:99 — "func publishedVersion(repoRoot string) string {"
    evidence: internal/core/launch/render.go:365 — "func stampMarketplace"
  - marketplace add REPPL/abcd — the README documents [redacted-user]/abcd
    evidence: README.md:83 — "/plugin marketplace add [redacted-user]/abcd"
- missing:
  - a bumped plugin.json.version and updated marketplace.json that /plugin update actually pulls — the stamped payload is staged only with --payload-dir and no workflow publishes it; the marketplace source is the unversioned git tree
    evidence: internal/surface/cli/ship.go:224 — "if payloadDir != "" && ingested.Written {"
    evidence: .github/workflows/release.yml:327 — "gh release create "${TAG}" bin/abcd-* bin/checksums.txt"
  - an explicit --version < x.0.0> override on the ship verb
    evidence: internal/surface/cli/ship.go:262 — "cmd.Flags().StringVar(&changelogJSON, "changelog-json""
  - a launch report naming the completed phase
    evidence: internal/surface/cli/ship.go:143 — "return string(cut.Impact) + ": " + strings.Join(cut.DecidedBy, ", ")"