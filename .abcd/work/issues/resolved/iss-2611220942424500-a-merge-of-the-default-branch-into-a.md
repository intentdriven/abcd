---
schema_version: 1
id: "iss-2611220942424500"
slug: "a-merge-of-the-default-branch-into-a"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "autonomous fix run 2026-11-21/22"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/landing"
remedy: "Let --rests-on accept a delta whose only commit is a merge of origin's default branch that git resolved with no hand edit (an empty remerge-diff) and whose incoming side touches no file the branch changes, since what main brought was reviewed when it landed; the merge gate checks the same at landing."
resolution: "A delta passes over a clean merge of origin's tracked default branch only when the merge has one merge base and its incoming side touches no file the branch's own commits change; the record path and the merge gate judge it through one function."
impact: fix
---

A merge of the default branch into a reviewed branch moves its head, so the review pair's receipt no longer names it, and a delta review (--rests-on) is refused because what main brought is not docs-only. On 22 Nov 2026 a bundle needed a whole new review pair after merging a main that added glossary text and one lint test, though the merge resolved with no hand edit and touched nothing the branch changed.
