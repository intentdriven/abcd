---
term: release
bounded_context: distribution
definition: A published, version-tagged snapshot of abcd — the act and the artefact of cutting a curated release from the single repo, carrying a version and a changelog entry.
aliases: ["published snapshot", "plugin release", "snapshot"]
forbidden_synonyms: []
status: stable
introduced_in: itd-67
starts_when: null
ends_when: null
not_to_be_confused_with: [distribution/version, core/record-families]
versions: null
---

# release (distribution)

A **release** is a published, version-tagged snapshot of abcd — both the act of
cutting a curated release from the single repo
([adr-28](../../../decisions/adrs/0028-single-repo-curated-release.md)) and the
resulting artefact, a GitHub Release whose published binaries do not carry
`.abcd/**` (the packaging filter's structural deny, which no cut release has
run yet), carrying a [version](version.md), a changelog entry, and a git tag.

As with [version](version.md), [phase](../core/phase.md) forbids "release" as a
synonym *in the core context* — abcd does not organise development by releases.
But in the distribution context, "release" is the correct term for the published
output: a phase completing may TRIGGER a release (a minor version bump), but the
release is the publish event, not the sequencing unit.

## When to use

Use "release" for a published, version-tagged snapshot cut from the repo, and for
the act of publishing one via `launch ship`.

## When NOT to use

Do not use "release" for an internal stretch of development work (that is a
[phase](../core/phase.md)) or as a loose synonym for "milestone" (the phase's end
condition).

## Examples

- "The v0.2.0 release publishes the completed launch phase as a GitHub Release
  cut from the repo."
- "`launch ship` refuses a no-change release unless `--force` is passed."

## Related terms

- [record families](../core/record-families.md) — the one page that maps the record families and how they relate
- [version](version.md) — the semver string a release carries
- [phase](../core/phase.md) — completing one may trigger a minor-version release
