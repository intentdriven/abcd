---
term: version
bounded_context: distribution
definition: A strict-SemVer string stamped into the curated release artifact at cut time and carried as the git tag of the single repo, identifying a published snapshot of abcd for install and update. It is an OUTPUT of publishing, derived from the impact of what shipped, and never a unit that sequences the work.
aliases: ["semver", "plugin version"]
forbidden_synonyms: []
status: stable
introduced_in: itd-67
starts_when: null
ends_when: null
not_to_be_confused_with: distribution/release
versions: null
---

# version (distribution)

A **version** in the distribution context is the semantic-version string that
identifies a published snapshot of abcd — stamped into the curated release
artifact at cut time and carried as the git tag of the single repo
([adr-28](../../../decisions/adrs/0028-single-repo-curated-release.md)), so the
host can compare what is installed against what is available and users can
update. The working tree stays unversioned; the version lives only in the cut
artifact and its tag — the dev-unversioned / release-versioned polarity applied
within one tree.

A version never sequences the work: the order of work is dependencies plus the
lifecycle shelves ([record families](../core/record-families.md),
adr-2609212115255771), and nothing is planned into a version. When abcd is
PUBLISHED, a semantic version is the precise, correct term: `launch ship` derives
it from the impact the shipped records declare (pre-1.0, a breaking change bumps
the minor), and an all-internal or empty cut derives none.

## When to use

Use "version" for the semver string of a published abcd snapshot — in the release
artifact, the git tag, the marketplace entry, and the changelog.

## When NOT to use

Do not use "version" for the sequencing of development work: that is dependencies
and the shelves. A version is the output of publishing, never the unit that
organises what ships together; the delivery grouping is a
[bundle](../core/bundle.md).

## Examples

- "`launch ship` stamps the release artifact `0.2.0` (strict SemVer, no leading `v`) and tags the repo `v0.2.0`."
- "Users update to the latest version with `/plugin update abcd`."

## Related terms

- [record families](../core/record-families.md) — the one page that maps the record families and how they relate
- [phase](../core/phase.md): the retired sequencing unit; a version was never one
- [release](release.md) — the published act that carries a version
