---
id: itd-66
slug: launch-payload-render-parity
spec_id: null
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
- All read-only w.r.t. the dev repo (temp-tree writes only, removed after) — matches the side-effect-free posture of the spc-64 gate.
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
