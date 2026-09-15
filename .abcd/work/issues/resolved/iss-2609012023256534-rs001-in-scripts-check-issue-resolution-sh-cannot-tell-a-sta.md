---
schema_version: 1
id: "iss-2609012023256534"
slug: "rs001-in-scripts-check-issue-resolution-sh-cannot-tell-a-sta"
severity: "minor"
category: "ux"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
resolution: "RS001 now says what the script can prove about a trailer it cannot satisfy. A record already terminal at the base ref is told apart by its base-side history: placed there by a base-side commit the head lacks, the message names that commit and the behind-count and prescribes a rebase (dropping the trailer only if the commit survives it); terminal before the branch diverged, it says the trailer names an issue resolved before this commit and prescribes dropping it, with no rebase. A record absent from the head tree while the base holds it names the base's folder and a rebase; an id with no record anywhere says so and prescribes checking the id. Every shape stays a refusal, because a rebase makes the range honest and the merged commits vanish from it. Four cases in check-issue-resolution-cases.sh assert the diagnosis text as well as the exit code, and two more assert the rebase is not prescribed where it cannot help; all four positive cases failed against the previous script."
impact: fix
---

RS001 in scripts/check-issue-resolution.sh cannot tell a stale branch from a forgotten resolution, and its message diagnoses the wrong one. On a branch 235 commits behind origin/main the pre-push preflight emitted 84 RS001 violations, every one instructing the reader to resolve an issue that already sat in a terminal folder at the base: the branch's own commits had been squash- or rebase-merged, so origin/main held the moved records, the two-dot diff of the two trees showed no record entering resolved/ or wontfix/, and the trailers went unsatisfied within the range. No line of the output said the branch was behind, and the one remedy that works, a rebase, appeared nowhere. The gate should distinguish the shapes it can prove: a record already terminal at the base ref (with how far behind the head is and which base-side commit placed it there), a record absent from the head tree while the base holds it, and an id with no record anywhere, and name a rebase as the likely remedy where the evidence points to one. Carried from the session handover into autonomous-run-2026-09-01.

## Grounds

- pursued: the reader acts on the diagnosis, not the exit code, so the message must name the shape the evidence supports; if a stale branch still produces a wall of resolve-it-here messages, the base-side history probe missed the placing commit, and if a rebase is prescribed where the record was terminal at the merge base, the ordering of the shapes is wrong
