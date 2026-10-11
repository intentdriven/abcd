---
schema_version: 1
id: "iss-183"
slug: "abcd-never-ships-claim-survives-in"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "2026-08-05 iss-43 review"
found_at: ".abcd/work/CONTEXT.md"
resolution: "Reword direction taken, not wiring: the descriptive instances now carry the channel-truthful phrasing the README adopted under iss-43 — `.abcd/**` is present in every repository checkout, marketplace installs and release source archives included, and never in the released binaries — with the launch bundler named as the implemented structural deny that no cut release has run. Reworded: .abcd/work/CONTEXT.md sharp-edges bullet, AGENTS.md (the boundaries bullet and, found by a case-insensitive re-enumeration the issue body's list missed, the working-tree-layout line), .abcd/README.md, .abcd/development/README.md, brief/02-constraints/01-platform.md, brief/05-internals/03-configuration.md record row, brief/01-product/02-context.md (both instances), brief/glossary/distribution/release.md, roadmap/phases/phase-1-ahoy.md, and the internal/adapter/scanner/network.go persona-registry comment. Classified exempt and left: brief/01-product/01-press-release.md, whose genre is the intended product stated in press-release voice (its unshipped claims stand unqualified throughout, so singling this one out would be inconsistent); brief/04-surfaces/04-launch.md, whose mention sits inside explicit 'The full-cut design' framing and is already truthful; plus the pre-declared exempt set — adr-0028, the dated plans, planned/shipped intent bodies, 03-mental-model.md and principles/script-first-mvp.md (different 'ships' claims), and launch/bundle_test.go (describes the deny mechanism itself). Wiring the launch publish path so the filter runs on a real release stays open feature work."
impact: fix
---

The blanket `.abcd/` exclusion claim that iss-43 corrected in the README survives across the record. Recorded rather than fixed there, because the disposition scoped iss-43 to one README line.

What is true: the exclusion is **implemented but never runs on a release**. `internal/core/launch/bundle.go:27-31` declares `DenyNamespaces` — "first-path-segment names that never ship", `.abcd` among them — as a structural deny no allowlist overrides, and `bundle_test.go` pins it (`TestAbcdNamespaceStructurallyExcluded`, `TestDefaultDenyNewTopLevel`). The wired `abcd launch ship` does exercise the deny, through the render path rather than the one its name suggests: `launch.PrecheckPayload` resolves the bundle (`internal/core/launch/render.go:192`) and `launch.RenderPayload` goes through it, both reached from `internal/surface/cli/ship.go` — the precheck under `--payload-dir`, which guards that call (`internal/surface/cli/ship.go:206-208`). `abcd launch --dry-run` exercises it unconditionally on a second path, `launch.DryRun` resolving the bundle directly (`internal/core/launch/dryrun.go:57`, from `internal/surface/cli/cli.go:112`). Separately, `launch.Ship` — the would-publish stub that "stops HERE and returns WouldPublish=true with NO network call" (`internal/core/launch/ship.go:36-41`) — has no production caller at all; the only calls are in `dryrun_test.go`. And `.github/workflows/release.yml` invokes neither: it builds the four binaries and uploads them with `checksums.txt` directly. So no release the project has cut has passed through the filter, and two channels carry `.abcd/` regardless: a marketplace install takes the repository root (`.claude-plugin/marketplace.json`, plugin `source: "./"`), and GitHub attaches an auto-generated source archive to every release, `.gitattributes` declaring no `export-ignore`.

The defect is therefore narrower than "the claim is false" and sharper than "the docs are stale": prose written in the present tense describes a mechanism that exists in code but has never run on a release.

Instances — descriptive and orientation documents asserting the claim as present-tense operating fact. Line refs rot; the quotes anchor.

1. `.abcd/work/CONTEXT.md:42` — "Single repo, curated release (no dev→public mirror). `.abcd/**` never ships." Sharpest of the set: CONTEXT.md is the first file a session reads, and a sharp-edges bullet is trusted before anything else is.
2. `AGENTS.md:112-113` — "**Single repo, curated release.** `.abcd/**` stays in-tree but is excluded from the release artifact by packaging; the repo is the plugin marketplace."
3. `.abcd/README.md:4` — "repo (transparent) but is excluded from the release artifact."
4. `.abcd/development/README.md:4` — "repo (transparent) but excluded from the release artifact".
5. `.abcd/development/brief/02-constraints/01-platform.md:9` — "**`.abcd/**` stays in-tree but is excluded from the release artifact by packaging** — exclusion is a build-time filter over the one tree, not a copy between two repos."
6. `.abcd/development/brief/05-internals/03-configuration.md:209` — the record row's `Committed?` cell: "committed — excluded from the release artefact by packaging".
7. `.abcd/development/roadmap/phases/phase-1-ahoy.md:11` — "`.abcd/**` excluded by packaging so the design record never ships in the" (the sentence continues on the following line).

Exempt, deliberately left (iss-42 and iss-44 precedent — a decision record and a dated snapshot are supposed to say what was decided then, and rewriting them to match today falsifies the record): adr-0028, which ratifies the wording; the dated plans under `.abcd/development/plans/`; and the bodies of intents under `planned/` and `shipped/`, which specify the mechanism they were written to deliver.

This list is not exhaustive and must not be trusted as one. The implementing round re-enumerates first, over the whole tree, with `never ships`, `excluded from the release`, and `by packaging` — the survey behind this issue already surfaces further candidates in `brief/01-product/`, `brief/04-surfaces/04-launch.md`, `brief/glossary/distribution/release.md` and `principles/script-first-mvp.md`, unclassified here — and sorts each hit into the descriptive set or the exempt set before touching anything.

Fix direction, one of two, chosen once and applied consistently: wire the existing publish path so the filter runs on a real release, which makes the prose true as written; or reword the descriptive instances to the current gap, on the same channel-truthful pattern the README now carries — what every repository checkout holds against what the released binaries hold — describing the launch bundler as the implemented mechanism it is rather than an operating one. Wiring is feature work with its own design questions and is not assumed here.
