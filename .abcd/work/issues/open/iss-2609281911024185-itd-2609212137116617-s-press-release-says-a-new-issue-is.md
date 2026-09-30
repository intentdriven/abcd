---
schema_version: 1
id: "iss-2609281911024185"
slug: "itd-2609212137116617-s-press-release-says-a-new-issue-is"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-2609212137116617"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/report/inbox.go"
deferred_after: "v0.11.1"
deferral_reason: "a lane of its own, or a ruling: routing inbox promote, IngestConsistency and IngestReading through the filing-time match threads the layered match config through the report package and moves the two ingests behind matchAndLink, more than an hour with tests; the alternative, narrowing itd-2609212137116617's press release to the two verbs, is the product thinker's text to change."
remedy: "Route every ledger writer through the filing-time match: inbox promote passes the layered match config on CaptureRequest.Match (internal/core/report/inbox.go captureRequest), and IngestConsistency and IngestReading write through matchAndLink (internal/core/capture/workflow.go), proven by one test per writer that files a near-double and finds the duplicates or refines link written."
---

itd-2609212137116617's press release says a new issue is matched against the record at filing, but only the capture verb and the quoted-text intent create run the filing-time match. The ledger's other writers file without it: inbox promote builds a capture.CaptureRequest with no Match (internal/core/report/inbox.go captureRequest), and IngestConsistency and IngestReading (internal/core/capture/consistency.go, reading.go) write issue records outside matchAndLink (internal/core/capture/workflow.go). The unattended paths the intent's Grounds name as the reason for the match are exactly the ones that skip it, so a double promoted from a peer report or filed by a consistency pass is never linked at filing. Either these writers pass the layered match config through CaptureRequest.Match, or the record narrows its claim to the two verbs.

## Remedy grounds (2026-09-29)

This keeps itd-2609212137116617's shipped promise instead of narrowing it, and the unattended writers are the ones the intent's Grounds name as the reason for the match. Rejected: narrowing the press release to the two verbs, which is the product thinker's text and would leave the doubles unlinked.

## Progress (2026-09-30)

Two of the three routes landed on branch feat/filing-duplicate-every-route: inbox promote and the consistency ingest now run the filing-time match through CaptureRequest.Match, on the report's own title and prose and on the finding's summary and explanation. The reading-ingest route waits on ruling DQ2b, so this record stays open.
