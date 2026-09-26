---
schema_version: 1
id: "iss-2608230752354927"
slug: "relative-links-under-abcd-work-are-checked-by-no-lint-root"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "record-review"
found_at: ".abcd/record-lint.json"
details: "record-lint's links_resolve is a blocker rule but its roots are ['.abcd/development'], and docs-lint's roots are ['docs','README.md']. Nothing lints .abcd/work/, so a broken relative link in an issue, a review record, DECISIONS.md, or CONTEXT.md is never reported. Some record-lint rules (issue_id_unique, issue_impact_valid) do reach .abcd/work/issues by explicit config, so the tier is partly covered and the gap is easy to mistake for coverage."
suggested_fix: "Either add .abcd/work to record-lint's roots (and check the blast radius on the rules that would newly apply to it), or scope links_resolve to its own path set covering both tiers. Prefer the second if the other rules are not wanted there."
related_issues: []
resolution: "links_resolve gains extra_roots, walked for links alone; the shipped record-lint config names .abcd/work and exempts the append-only reviews charter. The eight dead links it found in resolved records are repaired. TestLinksResolveWalksExtraRoots and TestLinksResolveCoversTheWorkTierInRealConfig pin it."
impact: internal
resolved_by:
  commit: "619b9521"
---

relative links under .abcd/work/ are checked by no lint root

`links_resolve` is a blocker, but blockers only fire inside a configured
root:

- record-lint `roots`: `.abcd/development`
- docs-lint `roots`: `docs`, `README.md`

`.abcd/work/` is in neither, so no relative link in the working tier is ever
resolved. That covers the issue ledger, the reviews charter, `DECISIONS.md`,
and `CONTEXT.md`.

Confirmed incidence, not theory: the first draft of
iss-2608221457227162 used `../../development/...` for two ADR links. From
`.abcd/work/issues/open/` the correct depth is three levels, so both were dead
on arrival. `make record-lint` and `abcd docs lint` both exited 0. The links
were fixed by hand after a manual check.

The partial coverage is what makes this easy to miss: `issue_id_unique` and
`issue_impact_valid` are configured with explicit `.abcd/work/issues` paths, so
the ledger visibly *is* linted, just not for links.

## Grounds

- pursued: a dead relative link anywhere in .abcd/work outside reviews/ fails record-lint; one that passes would show it wrong
