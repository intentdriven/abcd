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
---

itd-2609212137116617's press release says a new issue is matched against the record at filing, but only the capture verb and the quoted-text intent create run the filing-time match. The ledger's other writers file without it: inbox promote builds a capture.CaptureRequest with no Match (internal/core/report/inbox.go captureRequest), and IngestConsistency and IngestReading (internal/core/capture/consistency.go, reading.go) write issue records outside matchAndLink (internal/core/capture/workflow.go). The unattended paths the intent's Grounds name as the reason for the match are exactly the ones that skip it, so a double promoted from a peer report or filed by a consistency pass is never linked at filing. Either these writers pass the layered match config through CaptureRequest.Match, or the record narrows its claim to the two verbs.
