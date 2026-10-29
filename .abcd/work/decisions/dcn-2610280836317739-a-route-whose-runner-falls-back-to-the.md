---
id: dcn-2610280836317739
slug: a-route-whose-runner-falls-back-to-the
date: 2026-10-28
---
2026-10-28 — A --route whose runner falls back to the host is not recorded on the fallback receipt; the build brief and the implement page now say so, instead of the fallback receipt gaining an override field. Grounds: the fallback receipt already names the runner asked, the role then ran on the host so no route ran it, and adding a field would need its own run-state refusal case for version 15 and earlier for a reading nothing yet uses. The drain's route record carries the override field only because it shares the route record with the run state; no drain flag sets it, and the drain brief and drain.go now say that (review of the route-r2 branch).
