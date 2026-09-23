---
id: adr-20
slug: manifest-version-lockstep
status: accepted
date: 2026-07-03
supersedes: null
superseded_by: null
related_intents: [itd-69]
related_rfcs: []
related_adrs: [adr-19, adr-28, adr-2609231048308186]
---

# ADR-20: The two release manifests stay version-consistent by an anti-drift checker over a pinned path list; the source view stays unversioned; `--allow-dirty` must never bypass it

## Context

[adr-19](0019-plugin-json-version-carve-out.md) settled **where** the plugin
version lives: only in the curated release artifact, at the location the
version-location decision artifact
(`.abcd/config/version-location.json`)
selects — today `plugin.json` at pointer `/version`. adr-19 owns the *write*
location and the source-stays-unversioned polarity.

What adr-19 does NOT settle is the **anti-drift invariant** between the two
manifests. The version is single-sourced by the manifest render step (driven by
`version-location.json`), and the marketplace metadata
(`.claude-plugin/marketplace.json`) records the same release's version and
changelog entry. Because two files describe one release, a broken or partial cut
can leave a **half-state** — a version in one manifest and not the other, or a
`present-null` where an absent key was expected. `version-location.json` alone
encodes only the *selected primary location*; it does not enumerate the *full
set* of version/changelog-bearing fields the consistency check must inspect. That
enumeration is the gap this ADR closes. The repo is the marketplace
([adr-28](0028-single-repo-curated-release.md)): both manifests live in the one
tree, and the release view is the cut that carries versions.

Three coupled questions had to be settled:

1. **What exactly is checked, per view?** The source view and the release view
   have opposite polarity (adr-19: source unversioned, release versioned). A
   heuristic "find every version-bearing field" scan is brittle and untestable.
   We need an explicit, pinned path list the checker reads — never discovered.
2. **Does `--allow-dirty` bypass the check?** The launch pre-flight exposes an
   `--allow-dirty` escape hatch for a dirty worktree. Manifest consistency is a
   correctness invariant of the *released output*, not a worktree-cleanliness
   concern, so `--allow-dirty` must not weaken it.
3. **What shape is a marketplace changelog entry?** "Records the version +
   changelog entry in the marketplace metadata" is otherwise under-specified. The
   publishing bump step is the stated consumer that writes and validates the
   entry, so the entry needs a machine-checkable contract.

## Decision

### The lockstep policy

The plugin version is **single-source-written** by the manifest render step
(driven by `version-location.json`); no other code writes a version into a
manifest. The anti-drift invariant — the two manifests, plus the
`version-location.json` contract, describe the same release consistently — is
proven by a **read-only checker** (a Go implementation, with its CLI). The
checker never writes, never bumps, and **has no bypass/skip flag at its own
layer**.

The source view stays **unversioned** (adr-19): the version keys the checker
inspects are expected ABSENT there. The release view is **versioned**: the
version at the contract location must exist and every pinned secondary location
must agree.

### The pinned per-view path list (the contract this ADR owns)

The checker reads this list; it never discovers version-bearing fields
heuristically. Each entry is `(file, RFC-6901 pointer, meaning)`. The primary
location is not hard-coded here — it is read from `version-location.json`
(`manifest_path` + `json_pointer`) so an adr-19 relocation is absorbed without
editing this list. The secondary locations ARE pinned here, because
`version-location.json` does not enumerate them.

**RELEASE view — every pinned location must carry the release version, and all
must AGREE:**

| # | File | Pointer | Meaning |
|---|------|---------|---------|
| R1 | *(from `version-location.json`)* `manifest_path` | *(from `version-location.json`)* `json_pointer` | The primary version location adr-19 selects (today `.claude-plugin/plugin.json` `/version`). |
| R2 | `.claude-plugin/marketplace.json` | `/plugins/0/version` | The marketplace plugin entry's published version — must equal R1. |
| R3 | `.claude-plugin/marketplace.json` | `/plugins/0/changelog` | The marketplace plugin entry's changelog location/entry (validated against the changelog-entry schema, see below); its `version` field must equal R1. |

**SOURCE view — every pinned location's version key must be ABSENT (adr-19
polarity):**

| # | File | Pointer | Meaning |
|---|------|---------|---------|
| S1 | *(from `version-location.json`)* `manifest_path` | *(from `version-location.json`)* `json_pointer` | The primary location must NOT carry a version in the source view. |
| S2 | `.claude-plugin/marketplace.json` | `/plugins/0/version` | The marketplace entry must NOT carry a version in the source view. |
| S3 | `.claude-plugin/marketplace.json` | `/plugins/0/changelog` | No changelog entry in the source-view marketplace. |

`present-null` (`"key" in data` but value `null`) and `absent-key` (`"key" not in
data`) are **distinguished**: the source view requires absent-key at the version
pointers (a `present-null` is drift, not compliance); the release view requires a
present non-null agreeing value.

### Exit codes and report format (pinned)

The checker's contract, consumed by the launch pre-flight wiring:

| Exit | Meaning |
|------|---------|
| 0 | consistent |
| 1 | drift detected |
| 2 | contract unreadable (a pinned file missing/unparseable, or `version-location.json` malformed) |

`stdout` carries a machine-parseable report: one `DRIFT <view> <file><pointer>:
<detail>` line per drifting field, an `OK <view>` line when consistent, and an
`UNREADABLE <detail>` line for exit 2. The `--view source|release` argument is
**explicit** — no auto-detection.

### `--allow-dirty` must NOT bypass manifest consistency (wiring policy)

When the launch pre-flight wires this checker in, the manifest lockstep gate MUST
run and MUST be honoured **even under `--allow-dirty`**. `--allow-dirty` relaxes
the worktree-cleanliness pre-flight, not the released-output-correctness
invariant. **This is a POLICY this ADR records.** The checker itself ships only
as the flag-less module (which structurally cannot be bypassed at its own layer)
plus this policy record. The rule becomes enforced when the launch pre-flight
lands its wiring test asserting the gate fires under `--allow-dirty`. Until then
it is **advisory**.

### The release-cut handoff

The publishing step is the caller. After it writes the manifests via the render
step and records the changelog entry, its bump step invokes the checker in
`--view release` mode against the staged release view and refuses to publish on a
non-zero exit. The publishing step also validates the changelog entry it writes
against the changelog-entry schema (below). Registering the gate in the
pre-flight suite is launch-pre-flight territory; this decision delivers the
callable module + CLI + tests and this handoff record.

### The marketplace changelog-entry schema

**FORM decision (the deciding factor is the publishing step's consumption
mode):** the bump step validates the changelog entry **programmatically** before
publishing (the stated consumer writes it from the git-tag + launch-report
auto-changelog and must prove its shape). Because the entry is machine-written
and machine-validated — not human-authored prose — the schema is a **committed
schema file** (`changelog-entry.schema.json`), not ADR-section prose. The
ADR-prose form is reserved for the human-authored-and-never-machine-validated
case, which does not apply here.

The changelog entry records, per published release: the `version` (must equal
R1), the bump `tier` (`patch|minor|major`), the human `reason` string, the
release `date`, and the `source_sha` the release was cut from. Its authoritative
field list lives in the committed schema; see the schema `description` for the
binding detail.

## Alternatives Considered

- **Heuristic version-field discovery.** Rejected: scanning every manifest for
  "version-looking" keys is brittle, untestable, and would silently miss a new
  version location or false-positive on an unrelated `version` field (e.g. a
  `$schema` URL). A pinned list is the testable contract.
- **Auto-detect the view from the manifest content.** Rejected: the source and
  release views share identical file names and opposite version polarity —
  content-based detection would be circular (it would need the very version state
  it is checking to decide which polarity to enforce). An explicit `--view`
  argument removes the ambiguity.
- **Let the checker also write/repair drift.** Rejected: writes belong to the
  single-source render step (adr-19). A read-only checker keeps the write
  authority in one place and makes the gate safe to run anywhere.
- **Changelog entry as ADR-section prose only.** Rejected under the stated
  criterion: the publishing step validates the entry programmatically, so a
  committed schema is required; prose could not be loaded by the bump step.

## Consequences

- The anti-drift invariant is machine-checkable and pinned; a half-state cut is
  caught before it ships. The check direction is correct per view without
  heuristics.
- A future adr-19 relocation of the primary version location is absorbed: the
  checker reads R1/S1 from `version-location.json`, so only the decision artifact
  changes, not this list. Relocating a *secondary* location (R2/R3/S2/S3) is a
  deliberate edit to this ADR's pinned table plus the checker.
- The launch pre-flight gains a callable gate and a recorded obligation: wire it
  in and honour it under `--allow-dirty`, proven by a wiring test.
- The publishing step gains a committed changelog-entry schema to validate
  against, closing the marketplace changelog under-specification.
- The launch documentation states adr-19's polarity directly: only the release
  view is versioned, the render leaves the source view's manifests untouched, and
  the anti-drift note points here.
- [adr-2609231048308186](2609231048308186-the-catalog-pins-the-latest-release-s-plugin-archive.md) amends this decision: the source view's catalog names one
  release by archive address and digest, rewritten by every ship, so it is no
  longer release-neutral. Its version keys stay ABSENT, and R2/R3 are proven in
  the staged payload rather than in the published archive, which leaves the
  catalog out.
