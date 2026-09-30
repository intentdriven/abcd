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

A release is the derived checkpoint, never a sequencing unit: nothing is planned
into one. The order of work is dependencies (`blocked_by`, `builds_on`) plus the
lifecycle shelves, rendered as the Now / Next / Later block
([record families](../core/record-families.md), adr-2609212115255771). `launch ship`
derives the release's version from the impact of what shipped since the last tag
and composes its changelog from the records that reached a terminal folder. The
one forward-looking line a release keeps is an intent's `target_release`, which
the preview and the cut report and never refuse on.

## When to use

Use "release" for a published, version-tagged snapshot cut from the repo, and for
the act of publishing one via `launch ship`.

## When NOT to use

Do not use "release" for a stretch of development work or for a plan of what ships
together: the order of work is dependencies and the shelves, and the delivery
grouping is a [bundle](../core/bundle.md). Do not use it for the retired
[milestone](../core/milestone.md): an intent's end condition is its acceptance
criteria.

## Examples

- "The release publishes, as a GitHub Release cut from the repo, every record that
  reached a terminal folder since the last tag."
- "An all-internal or empty cut derives no version, and `launch ship` writes no dated
  heading for it."

## Related terms

- [record families](../core/record-families.md) — the one page that maps the record families and how they relate
- [version](version.md) — the semver string a release carries
- [phase](../core/phase.md): the retired sequencing unit; a release was never one
