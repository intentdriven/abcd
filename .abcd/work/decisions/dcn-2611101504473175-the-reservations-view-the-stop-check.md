---
id: dcn-2611101504473175
slug: the-reservations-view-the-stop-check
date: 2026-11-10
---
2026-11-10 — The reservations' view the stop check reads is bound to the check's deadline through one body (iss-2611100548341325). The readers on its route take a small git value: its empty form runs the calls they make today, so the context-free view and its baseline entries stay as they are, and its bound form, built only by claim.ViewContext, runs each git under the check's context. Only the reading that writes nothing is bound; the fetch, the keeping of a first sighting and every reader that keeps state are left as they are. ViewContext answers the context's error whenever the context has ended by the time it returns, because the view folds several failures into empty answers; the stop check then uses none of that snapshot and marks the reading cut short. The guard hook still reaches the context-free view through the delete prover, so the baseline is not lowered by this change.
