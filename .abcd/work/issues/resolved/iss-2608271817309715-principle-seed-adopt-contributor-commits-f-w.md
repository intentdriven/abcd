---
schema_version: 1
id: "iss-2608271817309715"
slug: "principle-seed-adopt-contributor-commits-f-w"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "itd-84 decomposition of the pilot-note proposal (2026-08-27)"
found_at: ".abcd/development/research/notes/2026-08-27-security-advisory-handling-pilot.md"
resolution: "decided: the principle adopt-contributor-commits (the pilot's F-W) is filed under development/principles"
impact: internal
---

principle seed from the pilot's F-W (itd-149 decomposition, part 3): the authorship default for external contributions is to ADOPT the contributor's commit, preserving their contributor-graph authorship; re-authoring with a Reported-by trailer is the honest fallback only when there is no branch to adopt, and the ACKNOWLEDGEMENTS entry rides the fix in the same change. No existing principle covers the authorship default; file it under development/principles/ (or fold into CONTRIBUTING) as its own act.

## Grounds

- pursued: the authorship default for external contributions is a filed principle; shown wrong if that principle is withdrawn or contradicts the adopt-first default
