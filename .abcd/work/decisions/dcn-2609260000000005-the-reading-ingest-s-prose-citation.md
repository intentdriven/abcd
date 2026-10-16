---
id: dcn-2609260000000005
slug: the-reading-ingest-s-prose-citation
date: 2026-09-26
---
2026-09-26 — The reading ingest's prose-citation refusal departs from "every refusal past the identity point is recorded" (the rule `refuse()` in internal/core/reading/ingest.go enforces, from iss-2608311518250688): it is returned unrecorded, because a recorded refusal gives the run an outcome and `refuseARerun` would then refuse the same run re-worded, where the run left parked is ingested again once its prose describes the record rather than citing a missing id — the stance the verdict ingest takes. It keeps the rest of the refusal contract: it rolls back this run id's own half-landed records and stage as `refuse()` does, and runs no sweep of other runs' orphans. `04-surfaces/23-reading.md` states it (implementer of lane fix2-drainL, autonomous run A, on review-drainL's design point for iss-2609261835118276).
