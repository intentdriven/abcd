---
id: adr-19
slug: plugin-json-version-carve-out
status: accepted
date: 2026-07-01
supersedes: null
superseded_by: null
related_intents: [itd-67]
related_rfcs: []
related_adrs: [adr-5, adr-28, adr-2609231048308186]
---

# ADR-19: The plugin version lives ONLY in the released artifact; the working tree stays unversioned, and the version location is chosen by a schema-validated decision, not hard-coded

## Context

abcd ships as an installable, versioned Claude Code plugin through a curated
release artifact cut from the single repo
([adr-28](0028-single-repo-curated-release.md)). That raises two coupled
questions the framework has to settle before any manifest is written:

1. **Where does the version live?** Claude Code discovers a plugin by directory
   convention and reads its manifest metadata. A published plugin needs a version
   so `/plugin update abcd` can compare installed-vs-available. But
   [adr-5](0005-brief-is-current-state.md) establishes that the working record
   carries **no** version label — versions are an *output* of the release cut,
   not a sequencing input in the working tree. So the working tree's committed
   `.claude-plugin/plugin.json` stays unversioned while the *release artifact*
   carries the version.

2. **Is `version` even a legal field?** Whether the Claude Code plugin-manifest
   schema *accepts* a `version` property is an external unknown until it is
   validated. Hard-coding `plugin.json.version` as the answer would be a guess.
   The version-location decision resolves it against the pinned schemastore
   fixtures and records the outcome in a machine-readable decision artifact
   (`.abcd/config/version-location.json`),
   with three fully-specified terminal outcomes: **ACCEPT** (`version` is an
   explicit schema property → write `plugin.json.version`), **FALLBACK** (a
   different schema-valid explicit-property location), or **BLOCKED** (no
   schema-valid explicit version location → escalate).

The recorded outcome is **ACCEPT**: `version` is an explicit property of the
pinned plugin-manifest schema, so the version location is
`.claude-plugin/plugin.json` at pointer `/version`, seed `0.1.0`. The marketplace
`source` resolves from the repo root (`marketplace_source_resolution_base:
repo_root`, `marketplace_source_to_root: ./`), unblocked.

## Decision

We will carry the plugin version **only** in the curated release artifact, at the
location the version-location decision artifact selects — today
`plugin.json.version`. The working tree's committed manifests stay
**unversioned**. The version is written at release-cut time (`/abcd:launch ship`)
by the manifest render step, which reads `version-location.json` and writes the
version at the recorded `manifest_path` + `json_pointer` — never by parsing a
location string, and never from a hard-coded field name. When the decision records
`blocked: true`, the render refuses: there is no schema-valid location to write, so
a versioned release manifest cannot be produced and the escalation stands.

This is compatible with the Claude Code plugin schema: spc-77.1 validated `version`
as an explicit property of the pinned manifest schema (not merely permitted by
`additionalProperties`), so a version-carrying public manifest is schema-valid. The
decision artifact records the exact accepting schema clause in its
`version_property_clause` field —
`claude-code-plugin-manifest.schema.json#/properties/version` — so this
compatibility claim traces to the pinned schema pointer, not to prose.

## Alternatives Considered

- **Version the working-tree `plugin.json` too, and let the release cut copy it.**
  Rejected: it contradicts adr-5 (the working record carries no version) and would
  make the committed state drift with every routine cut. The version is an output
  of the release cut; it belongs to the artifact, not the source.
- **Hard-code `plugin.json.version` as the version location.** Rejected: whether
  the schema accepts `version` was an external unknown, and a hard-code would have
  silently shipped a schema-invalid manifest in the REJECT world. The
  decision-artifact indirection lets every downstream consumer (render, public
  writes, smoke) read a validated location without re-deriving it, and keeps the
  framework honest if a future schema revision moves or removes the field.
- **Store the version in `marketplace.json` only.** This is the FALLBACK branch,
  kept live in the decision schema but not selected — `plugin.json.version` is the
  more conventional, install-visible location and it validated as an explicit
  property, so ACCEPT wins.

## Consequences

- The version is single-sourced in the release artifact; the working tree never
  carries a version, so there is nothing to keep in sync on the source side
  (honours adr-5).
- Every version-writing capability (the manifest render step, the `launch ship`
  writer, the release smoke) is **version-neutral**: it reads
  `version-location.json` rather than assuming a field name, and **refuses** when
  `blocked: true`. A future schema change that relocates the version is absorbed by
  re-running the version-location decision, not by editing call sites.
- The release artifact is installable+versioned; `/plugin update abcd` gains a
  version to compare against. The lifecycle is explicit: the working tree =
  unversioned source of truth; the release artifact = versioned published output.
- A new obligation: the no-half-state lint asserts the terminology and docs
  describe the *selected* location (or the escalation text under BLOCKED), never a
  stale hard-coded `plugin.json.version` claim.
- [adr-2609231048308186](2609231048308186-the-catalog-pins-the-latest-release-s-plugin-archive.md) amends this decision: the catalog sources the plugin from the
  latest release's pinned archive instead of the tree, so the version reaches an
  install through that published archive. The working tree still carries no
  version key, and the `./` source this ADR's Context records holds only until
  the first pinned ship. The contract carries no `marketplace_source_to_root`
  key: nothing read it, and the pin would have made it false.
