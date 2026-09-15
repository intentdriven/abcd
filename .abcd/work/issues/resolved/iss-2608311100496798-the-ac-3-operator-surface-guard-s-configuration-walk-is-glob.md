---
schema_version: 1
id: "iss-2608311100496798"
slug: "the-ac-3-operator-surface-guard-s-configuration-walk-is-glob"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "delta review of the itd-184 fix commit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/regime_surface_test.go"
resolution: "the configuration walk enumerates every tracked .abcd json from the git index instead of globbing two directories, and the two written lists that survive are now stated plainly in the guard header and in spc-62"
impact: internal
resolved_by:
  intent: "itd-184"
  spec: "spc-62"
---

The ac-3 operator-surface guard's configuration walk is globbed at two non-recursive directories, so four committed JSON files abcd reads as configuration sit outside it: the persona registry, the release surface, the release-gate manifest and the branch-ruleset mirror. A regime key written into the persona registry leaves the guard green. Two sentences claim otherwise: the guard's own header says nothing is a written list and the guard cannot fall behind the surface it guards, and spc-62's residue paragraph says the walk reaches a file once one carries the key. The glob is a written list of two directories and it has already fallen behind by four files, so both sentences are false as written. A claim about behaviour that the code does not establish is the itd-195 class this very criterion was split to enforce.

## Grounds

- pursued: the claim is made true rather than narrowed, because a header sentence that outruns the code is the class ac-3 exists to catch; what would show this wrong is a configuration file abcd reads that is not tracked under .abcd, which the index enumeration would not see
