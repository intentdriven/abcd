---
id: itd-66
slug: launch-payload-render-parity
spec_id: spc-2609201955277614
kind: standalone
suggested_kind: standalone
reclassification_history: []
related_adrs: [adr-28]
prd_path: null
grill_session_id: 66d0f1de-0066-4a66-9c0d-000000000066
grilled_at: 2026-07-01
grilled_intent_hash: fff78a6fd27d6a402d2a41ab67923024d426bff40ae556ad67fedecf86ad21fa
glossary_terms_used:
- distribution/release
- distribution/version
- core/brief
- core/intent
- core/spec
warrants_assumed:
- "Shipped Python modules may have import-time side effects; the smoke cannot assume import purity."
- "The previously published release may be absent at first launch; parity treats that as all-added, not an error."
severity: critical
impact: additive
---

# abcd Renders The Exact Public Payload, Proves The Excludes Never Leak, And Smoke-Tests The Installed Surface Before Any Snapshot

## Press Release

> **`/abcd:launch` gains a real payload render with a leak-proof default-deny filter, a parity diff against the previously published release, and an installed-surface smoke test — so a maintainer sees EXACTLY what would be published, provably without `.abcd/` leaking, and with every shipped `/abcd:*` command, skill, and hook confirmed to resolve and import from the rendered snapshot.** The payload is the curated release excluding `.abcd/**` ([adr-28](../../decisions/adrs/0028-single-repo-curated-release.md)); the brief's § 2 payload manifest is default-deny (include-list + hard `.abcd/**` exclusion), but nothing yet MATERIALISES that manifest and proves the exclusions hold, diffs it against the last published release, or verifies the rendered plugin actually loads. This intent builds that: render → prove-no-leak → parity-diff → consumer-surface smoke, all read-only, so the release is trustworthy before promotion.

> "Before I publish the release I want to see the actual file list, be certain none of my development knowledge or flow state rode along, and know the plugin still works once it's just the shipped files," said a maintainer. "A preview I have to trust isn't enough — render it and prove it."

## Why This Matters

The pre-flight gate suite ([[itd-65-launch-preflight-gate-suite]]) decides whether the payload is CLEAN; this intent decides whether the payload is CORRECT and COMPLETE — the other half of a trustworthy release. adr-28 makes the wholesale `.abcd/**` exclusion deliberate packaging policy (the curated release artifact carries plugin code, never the `.abcd/**` project-knowledge store); but policy without a test is a hope. A materialised render with an asserted default-deny filter turns "we exclude `.abcd/`" into "we PROVE no `.abcd/**` path is in the rendered tree." The parity diff against the previously published release shows precisely what a snapshot would change (and catches accidental deletions under `clean`/`overlay` modes). The installed-surface smoke test catches the failure an in-repo test cannot see: a command/skill/hook that works in the full tree but is broken once only the shipped files remain. Together they make a release a verified operation, not a leap.

## What's In Scope

- A read-only payload RENDER: materialise the § 2 include-manifest into a temp tree, applying `.gitignore` patterns and the default-deny exclude set (`.abcd/`, `.specstory/`, `memory/`).
- A leak-proof assertion: the rendered tree contains ZERO `.abcd/**` paths, and honours the `.abcd/launch.allow` allowlist contract (never promotes any `.abcd/**` line, per adr-28).
- A parity diff between the rendered payload and the previously published release: added / changed / removed files, so the operator previews the exact snapshot delta before promotion.
- An installed-surface smoke test: from the rendered snapshot, load `plugin.json` + `marketplace.json` and assert every declared `/abcd:*` command, skill, and hook resolves, and every shipped Python entrypoint imports.
- All read-only w.r.t. the dev repo (temp-tree writes only, removed after) — matches the side-effect-free posture of the spc-64 (predecessor store) gate.
- Canonical payload resolution (grill Q3): itd-66's render is the SINGLE resolver of "the payload" (include-manifest + default-deny + `.gitignore` + symlink-resolve). [[itd-65-launch-preflight-gate-suite]]'s gate suite scans exactly this resolved output and never re-resolves — so render and gate can never disagree on what is being shipped. This makes itd-66 (render) a dependency of itd-65 (gate): render → gate.
- Layered leak defense (grill Q2): the render asserts no excluded PATH in the tree AND resolves symlinks (a payload symlink targeting `.abcd/` fails the assertion); embedded `.abcd`/`.flow` CONTENT that rode along inside a shipped file is caught by itd-65's secret/PII/identity content scan. Structural exclusion here; content cleanliness there.
- Parity baseline (grill Q1): the diff targets the previously published release at a configured ref (default: the latest release tag). An absent/empty prior release yields an all-added diff (valid first-launch); a wrong/missing configured baseline is a hard error, never a silent empty diff.
- Deep-smoke isolation (grill Q4/Q5): the deep installed-surface smoke imports every shipped Python entrypoint, resolves every declared command/skill/agent/hook, and renders each command's help/frontmatter (short of full behavioral invocation). Imports run in an ISOLATED SUBPROCESS (cwd = temp render tree, guarded/minimal env) so any module-level side effect lands in the throwaway tree, preserving the read-only-w.r.t.-dev-repo guarantee. This deep check is the tier [[itd-67-installable-versioned-plugin]]'s light smoke is later upgraded to call.

## What's Out of Scope

- The pre-flight security/PII/marker gate suite (that is [[itd-65-launch-preflight-gate-suite]] — this intent is RENDER + PARITY + SMOKE).
- Actually pushing / mirror-mode execution / version bump + marketplace changelog write (brief §§ 3–4 — the promotion act itself; a later intent or the ship graduation).
- Re-including any `.abcd/**` path into the payload — adr-28 forbids it; the render must enforce, not relax, that.
- A full end-to-end publish dry-run to a real remote — the smoke test loads the rendered tree locally, it does not clone/push.

## Scope Conditions

None stated.

## Delivery Status

Reviewed against the v0.9.0 tree on 2026-09-20. Delivered and cited: every
include root rendered and the record namespace structurally excluded
(`TestAbcdNamespaceStructurallyExcluded`, `TestNestedDeniedNamespaceExcluded`,
`TestBundleShipsEveryPluginSurface`); a denied-rooted include refused
(`TestAbcdCannotBeReincluded`); a symlink into the namespace refused
(`TestSymlinkToRepoRootDoesNotLeakDenied`); no residue in the dev repo
(`TestRenderPayloadLeavesSourceTreeUnversioned`); the gate consuming the
render's own resolution (`TestPayloadTreeImplementationsResolveIdentically`);
the light smoke tier resolving commands, skills and hooks. Moot: the
Python-import clause (nothing shipped imports) and the seeded first baseline
(the first release is manual by the brief's bootstrap exception). Open, and
scoped by spc-2609201955277614: the file-level parity diff and the deep smoke
tier.
Both delivered on 2026-09-25; the Decisions below cite them.

## Decisions

- **2026-09-25 — the Python-import clause of the smoke criterion (fourth) is
  moot, and the rest of it stands.** Source: spc-2609201955277614 (Summary,
  piece 2: "the original criterion's Python-import clause is moot (nothing
  shipped imports) and is recorded as such on the intent") and the Delivery
  Status review above (2026-09-20). The payload's include roots carry no Python
  and no other imported entrypoint (`hooks/` and `scripts/` ship shell), so
  "every shipped Python entrypoint imports" has nothing to assert; the criterion
  text is unchanged. Stands and is met: every declared command, skill and hook
  resolves (the light tier), and every declared command, skill and agent page
  LOADS, rendered in an isolated subprocess rooted at a materialised copy of the
  payload, so a broken page fails (`SmokeDeep` and `RenderPageHelp` in
  `internal/core/launch/deepsmoke.go`, the hidden `abcd launch smoke-pages`
  child in `internal/surface/cli/launch_deep.go`;
  `TestSmokeDeepCatchesAPageThatResolvesButDoesNotLoad`,
  `TestLaunchDryRunDeepSmokeRunsInAnIsolatedSubprocess`,
  `TestLaunchShipRunsTheDeepTierAndParity`). The tier is opt-in on the preview
  (`--deep-smoke`) and always on in the cut, per the spec's Approach.
- **2026-09-25 — how the parity criteria are met** (spc-2609201955277614 piece
  1). The third criterion: `PayloadParity` in `internal/core/launch/parity.go`
  lists every payload path added, changed or removed with its SHA-256, on every
  preview and in the cut (its JSON, its render and its pre-flight report)
  (`TestParityAgainstARenderAtTheTagReportsEveryChange`,
  `TestLaunchDryRunReportsTheParityDiff`). The seventh: no previous tag, or a
  tag that declared no payload, is all-added with the reason stated
  (`TestParityWithNoPreviousReleaseIsAllAdded`,
  `TestParityAgainstATagThatShippedNoPayloadIsAllAdded`); a baseline that cannot
  be read is a named refusal, never an empty diff, and a configured `--baseline`
  that is not a release tag exits 2 by name
  (`TestParityWithAnUnreadableBaselineIsANamedRefusal`,
  `TestLaunchDryRunConfiguredBaseline`). The fifth still holds with both pieces
  in: the render at the tag and the deep tier write only private temporary
  trees they remove.
- **2026-09-25 — the baseline is read from the disk unless the operator asks
  for the network.** The spec reads the previous payload "from the tag's release
  asset when present and from a fresh render at that tag otherwise";
  [adr-38](../../decisions/adrs/0038-implicit-checks-are-disk-only.md) lets the
  network answer only an explicit ask. So the default baseline is a fresh render
  at the tag in a private clone of the checkout's own objects, and
  `--fetch-baseline` (preview and cut) reads the tag's published plugin archive
  first: every fetch announced on stderr and named in the report, the archive
  refused unless the release's own `checksums.txt` vouches for its digest, and a
  release publishing no archive falling back to the render at the tag, saying so
  (`TestParityAgainstTheReleaseAssetVerifiesAndDiffs`,
  `TestParityReleaseAssetChecksumMismatchFailsClosed`,
  `TestParityReleaseAssetAbsentFallsBackToARenderAtTheTag`,
  `TestLaunchDryRunFetchBaselineReadsTheVerifiedReleaseAsset`). The two
  manifests the release stamps are compared as canonical JSON with their
  version keys removed, since the render re-marshals them and the version bump
  is the cut's own report; against an archive, the catalog the archive omits by
  construction is named as not compared.

## Acceptance Criteria

> _Given-When-Then per the itd-1 discipline._

- **Given** the § 2 include-manifest, **when** the payload is rendered, **then** the rendered tree contains every include root and ZERO paths under `.abcd/`, `.specstory/`, or legacy `memory/`.
- **Given** a `.abcd/launch.allow` line pointing at a `.abcd/**` path, **when** the render applies the allowlist, **then** that line is refused / never promoted (adr-28), and the render records the refusal.
- **Given** a rendered payload and the previously published release, **when** the parity diff runs, **then** it reports added/changed/removed files accurately, so the operator sees the exact snapshot delta.
- **Given** a rendered snapshot, **when** the installed-surface smoke test runs, **then** every `/abcd:*` command, skill, and hook declared in the manifest resolves and every shipped Python entrypoint imports — a broken shipped entrypoint FAILS the test.
- **Given** the whole render + parity + smoke flow, **when** it runs, **then** it makes no change to the dev repo (temp-tree only) and leaves no residue — including deep-smoke imports, which run in an isolated subprocess rooted at the temp tree.
- **Given** a payload symlink pointing into `.abcd/`, **when** the render's leak assertion runs, **then** it resolves the symlink and FAILS (a symlink is not an escape hatch around path exclusion).
- **Given** no previously published release (or an empty one), **when** parity runs, **then** it produces an all-added diff (first-launch) rather than refusing; given a wrong configured baseline, it errors clearly.
- **Given** the resolved payload manifest, **when** itd-65's gate suite runs, **then** it consumes THIS render's resolution and does not independently re-resolve the payload.

## Prior Art

- [adr-28](../../decisions/adrs/0028-single-repo-curated-release.md) — the packaging policy (`.abcd/**` never ships) this intent turns from policy into a proved assertion.
- [[itd-65-launch-preflight-gate-suite]] — sibling: its gate suite consumes this render's resolved payload and never re-resolves (render → gate).
- `spc-78-launch-payload-render-parity-smoke` — the predecessor implementation's spec for this contract, carried as design input per the brief's delivery-state provenance note. Two of its implementation decisions diverge from this repo's brief: its render consumed a `.abcd/config/launch-payload.json` config rather than the `.abcd/launch.allow` allowlist, and its smoke discovered surfaces by directory convention rather than manifest declaration. The brief (`04-surfaces/04-launch.md`) remains canonical here; the deltas are weighed at spec time (see Open Questions).

## Open Questions

- Predecessor delta (spc-78): adopt a config-file payload override (`.abcd/config/launch-payload.json`) and directory-convention smoke discovery, or keep the brief's `.abcd/launch.allow` + manifest-declared surfaces? Adjudicate at spec time; a brief change needs its own edit, not a silent divergence.
- Does the render reuse the walk/filter logic already in the native launch capability (its walk-root routine and the include/exclude sets), promoting it from preview-only to a real materialiser, or is it a fresh module the dry-run then reuses?
- What is the parity baseline when there is no previously published release or it is on a different ref — treat all-added, or require a configured baseline?
- How deep does the smoke test go — manifest resolution + import only, or also a minimal invocation of each command's help surface?
- Should the render assert against the SAME resolved manifest the pre-flight suite ([[itd-65-launch-preflight-gate-suite]]) scans, so the two never diverge on what "the payload" is?

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-04706634ba21 -->
Fidelity review — receipt rcp-04706634ba21 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:a64cc13441a073183a99f59e2760d86ac7ec242999c85bf400b4ef087c82525d
Input attestations: diff:internal/core/launch at main 811fba17 (git ls-tree -r; the render, parity and deep-smoke modules as delivered through 2026-09-25)@sha256:3a6ae83973dc3e803e7fa3c69deb981dc4b1790082324964db59f244d046210f;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the bundle's denied set names .abcd, .specstory and memory beside .git, .flow and .work, a test proves the namespace is structurally excluded at every depth, and another proves every plugin surface the manifests declare is in the bundle
  evidence: internal/core/launch/bundle.go:35 — "".git": {}, ".abcd": {}, ".flow": {}, ".work": {}, ".specstory": {}, "memory": {},"
  evidence: internal/core/launch/bundle_test.go:78 — "func TestAbcdNamespaceStructurallyExcluded"
  evidence: internal/core/launch/bundle_denyseg_test.go:10 — "func TestNestedDeniedNamespaceExcluded"
  evidence: internal/core/launch/payload_completeness_test.go:40 — "func TestBundleShipsEveryPluginSurface"
- ac-2 — MET_WITH_CONCERNS: an include naming a denied namespace is refused at load with the pattern named, and even a hand-forced include slice cannot promote a .abcd path; the concern is the name: the criterion says `.abcd/launch.allow`, and the delivered allowlist is the includes list of `.abcd/config/launch-payload.json`, which the canonical launch brief names, so the delta the intent's open question deferred to spec time was settled by the brief rather than by the spec
  evidence: internal/core/launch/includes.go:149 — "return preflight("include pattern names a denied namespace segment %q: %q", seg, pattern)"
  evidence: internal/core/launch/includes_test.go:70 — "func TestAbcdCannotBeReincluded"
  evidence: .abcd/development/brief/04-surfaces/04-launch.md:335 — "`.abcd/config/launch-payload.json`, over one tree. Everything not named is"
- ac-3 — MET: PayloadParity lists every path added, changed or removed against a render at the previous tag with its SHA-256, and the preview reports it
  evidence: internal/core/launch/parity.go:186 — "func PayloadParity(repoRoot string, bundle Bundle, in ParityInput) ParityReport {"
  evidence: internal/core/launch/parity_test.go:66 — "func TestParityAgainstARenderAtTheTagReportsEveryChange"
  evidence: internal/surface/cli/launch_parity_test.go:52 — "func TestLaunchDryRunReportsTheParityDiff"
- ac-4 — MET_WITH_CONCERNS: the light tier resolves every declared command, skill and hook and the deep tier loads every declared page in an isolated subprocess so a page that resolves but does not load fails; the Python-import clause is moot because nothing shipped imports, recorded on the intent on 2026-09-25 with the criterion text unchanged
  evidence: internal/core/launch/deepsmoke.go:373 — "func SmokeDeep(root string, run PageRunner) DeepSmokeReport {"
  evidence: internal/core/launch/deepsmoke_test.go:165 — "func TestSmokeDeepCatchesAPageThatResolvesButDoesNotLoad"
  evidence: internal/surface/cli/launch_parity_test.go:449 — "func TestLaunchShipRunsTheDeepTierAndParity"
  evidence: .abcd/development/intents/shipped/itd-66-launch-payload-render-parity.md:80 — "the Python-import clause of the smoke criterion (fourth) is"
- ac-5 — MET: the render leaves the source tree unversioned and the deep smoke runs in an isolated subprocess rooted at a materialised temporary copy that is removed
  evidence: internal/core/launch/render_test.go:94 — "func TestRenderPayloadLeavesSourceTreeUnversioned"
  evidence: internal/surface/cli/launch_parity_test.go:390 — "func TestLaunchDryRunDeepSmokeRunsInAnIsolatedSubprocess"
- ac-6 — MET: a symlink to the repository root inside the payload does not leak the denied namespace: the test plants .abcd/secret.txt and asserts the render fails to carry it
  evidence: internal/core/launch/bundle_test.go:177 — "func TestSymlinkToRepoRootDoesNotLeakDenied"
- ac-7 — MET: no previous release, or a tag that shipped no payload, is all-added with the reason stated; an unreadable baseline is a named refusal and a configured baseline that is not a release tag exits 2 by name
  evidence: internal/core/launch/parity_test.go:113 — "func TestParityWithNoPreviousReleaseIsAllAdded"
  evidence: internal/core/launch/parity_test.go:138 — "func TestParityWithAnUnreadableBaselineIsANamedRefusal"
  evidence: internal/core/launch/parity.go:173 — "func ValidateBaselineTag(repoRoot, tag string) error {"
  evidence: internal/surface/cli/launch_parity_test.go:83 — "func TestLaunchDryRunConfiguredBaseline"
- ac-8 — MET: the two payload-tree implementations the render and the gate walk are proved to resolve identically over the repository, so the gate reads the render's resolution and cannot disagree with it
  evidence: internal/core/launch/installsurface_test.go:263 — "func TestPayloadTreeImplementationsResolveIdentically"

Gap audit:
- honoured:
  - render, prove-no-leak, parity diff and installed-surface smoke, all read-only with respect to the checkout
    evidence: internal/core/launch/render_test.go:94 — "TestRenderPayloadLeavesSourceTreeUnversioned"
    evidence: internal/core/launch/parity.go:186 — "func PayloadParity("
  - the baseline is read from disk unless the operator asks for the network, and a fetched archive is refused unless the release's own checksums vouch for it
    evidence: .abcd/development/intents/shipped/itd-66-launch-payload-render-parity.md:112 — "the baseline is read from the disk unless the operator asks"
    evidence: internal/core/launch/parity.go:268 — "func (rep *ParityReport) fillFromAsset("
- diverged:
  - the allowlist is `.abcd/launch.allow`; delivered as the includes list of `.abcd/config/launch-payload.json`, which the canonical brief names, so the predecessor delta was settled by the brief and not by a spec-time adjudication
    evidence: .abcd/development/brief/04-surfaces/04-launch.md:18 — "(`.abcd/config/launch-payload.json`) has nothing to preview, and the preview"
    evidence: .abcd/development/intents/shipped/itd-66-launch-payload-render-parity.md:152 — "Predecessor delta (spc-78)"
  - every shipped Python entrypoint imports; nothing shipped is Python, so the clause is moot by the intent's own 2026-09-25 decision
    evidence: .abcd/development/intents/shipped/itd-66-launch-payload-render-parity.md:80 — "the Python-import clause of the smoke criterion (fourth) is"
- missing: (none)
