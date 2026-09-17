---
schema_version: 1
id: "iss-2609081941074556"
slug: "releaseof-substring-match-misattributes-the-feature-stamp"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
related_issues: ["iss-2609090951280114", "iss-2609090951287232"]
resolution: "releaseOf now matches the record handle at a word boundary: every handle on a changelog line is read out through the package's own bodyHandleRe and compared whole, so itd-199 can no longer be credited by a newer section's itd-1990. Newest-section-first is unchanged. Covered by a synthetic superstring detector and an anti-vacuity guard on the committed CHANGELOG, where itd-1 was stamped 0.7.1 off the itd-130 credit and now takes 0.2.0, the release that names it." # <!-- record-lint: illustrative -->
impact: fix
---

releaseOf stamps the homepage feature block by strings.Contains of the record id against each CHANGELOG line, with no word boundary, walking newest dated section first. A short id therefore inherits a longer one: on ec7f40d6 releaseOf(itd-9) returns 0.4.1 off the itd-93 credit and releaseOf(itd-1) returns 0.7.1 off itd-130, and a future superstring landing in a newer section restamps the featured id. site check does not validate the stamp. The same package already carries bodyHandleRe. Fix: match the record id at a word boundary (reuse bodyHandleRe, or split the line into tokens); newest-section-first can stay. Detector: a changelog whose newer dated section mentions itd-1990 must not stamp featured itd-199, while a line naming itd-199 as its own id still matches. Independent of the Audit Notes rollup scanner. Reported as GitHub issue 624 against ec7f40d6. <!-- record-lint: illustrative -->

## Grounds

- pursued: the misattribution is entirely in the boundary of the match, so a whole-handle comparison fixes every case without changing which section wins; a featured record whose stamp is still wrong after this, or a stamp that disappears for a record the changelog does credit, would show it wrong.
