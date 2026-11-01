---
schema_version: 1
id: "iss-2611011155020210"
slug: "a-practice-run-dry-run-of-reading-ingest"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "intent audit of itd-2610301428390406, run-2610312053380475 lane 2, 2026-11-01"
origin: researcher-authored
production_mode: hand-written
found_at: "the practice runs of reading ingest and scribe ingest"
remedy: "Move the capture writer's validation (ordering gate, schema validation, size cap) into a plan stage that reading ingest's and scribe ingest's practice runs call, so the practice run refuses what the write would refuse. Make the refusal-recording paths word the practice refusal as the real one, with the recording location on a separate line. Add a plugin archive to the tree-wide test's launch ship fixture so the archive-pin and staged-payload branch runs under --dry-run."
---

A practice run (--dry-run) of reading ingest or scribe ingest can pass where the real write refuses. The capture verbs' own checks (the ordering gate, record schema validation and the size cap) run only when the record is written, and the scribe's plan stage says so (ledgerPending, internal/core/scribe/plan.go, landing with lane 2 of run-2610312053380475). So for these two commands the practice run's validator is narrower than the write's, which falls short of itd-2610301428390406's promise that a practice run shares every check. Two smaller gaps go with it. Past the point where reading ingest records a refusal, and in launch ship's payload path, the practice run's refusal names where the refusal would be recorded, so it is not the real run's message word for word. And the tree-wide dry-run test's launch ship fixture declares no plugin archive, so the archive-pin and staged-payload branch under --dry-run never runs. The intent auditor reported all three as concerns (MET_WITH_CONCERNS) on that lane (audit rcp-997517c0f4ba).
