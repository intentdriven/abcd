---
schema_version: 1
id: "iss-2610160957404827"
slug: "abcd-drain-reads-the-issue-ledger-of-the"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "drain check from the primary checkout, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Before a run, and in the dry run's output, abcd drain compares the checkout with origin's default branch and refuses (or warns in --dry-run) when it is behind, naming the command that updates it"
related_intents: [itd-2610161016394967]
resolution: "Each drain move fetches origin and plans from one snapshot of the ledger on origin's default branch, which the start check, the remedy digest and the footprint read too; the dry run plans from that branch as last fetched and names the view's age, the commits behind and the issues finished there; a reservation of a record finished on that branch backs off"
impact: fix
resolved_by:
  intent: "itd-2610161016394967"
  spec: "spc-2610161044461448"
---

abcd drain reads the issue ledger of the checkout it runs in and never checks whether that checkout is behind origin's default branch: from a primary checkout 185 commits behind main, the dry run listed 162 eligible issues from the stale ledger, many since fixed or changed, and a run would judge, reserve and work from that list.

Reproduced in a lab with three accounts on one origin (2026-10-16): with one account's checkout a commit behind after another resolved the head issue on origin, the behind account's dry run still listed it eligible, its drain reserved it (reserve makes no check that the record is resolved on origin's default branch, and the new reservation reads released at once), and opened a lane cut from origin's main, where the brief refused because the issue is already resolved there, advising to land the issue first. Every later move of that drain then answered "drive" the stalled lane (iss-2610161015368630). A drain never updates its own checkout, so on a busy day each drain falls further behind and this is reached within hours.

A fuller remedy: plan from the origin default branch's tree fetched at the start of each move, as the lane's brief already reads its base, and have a fresh reservation over a record finished on origin's default branch refuse or hand back (grounds: a queue consumer reads the row it locks, as SELECT ... FOR UPDATE SKIP LOCKED).

## Grounds

- pursued: a behind checkout's drain neither reserves, judges nor opens a lane for an issue resolved on origin; shown wrong if a drain on a stale checkout reserves an issue already resolved on origin's default branch
