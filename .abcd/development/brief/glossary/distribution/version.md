---
term: version
bounded_context: distribution
definition: A strict-SemVer string stamped into the curated release artifact at cut time and carried as the git tag of the single repo, identifying a published snapshot of abcd for install and update. It is an OUTPUT of publishing, distinct from the internal sequencing unit (phase).
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

This is deliberately distinct from [phase](../core/phase.md), which forbids
"version" as a synonym *in the core context*. That prohibition is correct there:
abcd organises its internal development work by phase, not by version number. But
when abcd is PUBLISHED, a semantic version is the precise, correct term — it is
what "falls out" when a phase (or a routine snapshot) is cut as a release. The
two coexist: `phase` sequences the work; `version` labels the published result.

## When to use

Use "version" for the semver string of a published abcd snapshot — in the release
artifact, the git tag, the marketplace entry, and the changelog.

## When NOT to use

Do not use "version" for the internal sequencing of development work — that is a
[phase](../core/phase.md). A version is the output of publishing, never the unit
that organises what ships together.

## Examples

- "`launch ship` stamps the release artifact `0.2.0` (strict SemVer, no leading `v`) and tags the repo `v0.2.0`."
- "Users update to the latest version with `/plugin update abcd`."

## Related terms

- [record families](../core/record-families.md) — the one page that maps the record families and how they relate
- [phase](../core/phase.md) — the internal sequencing unit; a version is an output of completing one
- [release](release.md) — the published act that carries a version
