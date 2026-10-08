---
schema_version: 1
id: "iss-2610080243367274"
slug: "testeverystreamfailureisrefusedwithoutthekey-a-stream-that"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "CI for PR #859, 2026-10-08"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/stream_test.go"
remedy: "Make TestEveryStreamFailureIsRefusedWithoutTheKey's never-ending-stream case drive the adapter's own deadline from a test clock or a short configured timeout and wait for it with a margin derived from that timeout, rather than a fixed 30s wall-clock wait that a loaded runner can exceed."
---

TestEveryStreamFailureIsRefusedWithoutTheKey/a_stream_that_never_ends failed on the CI ubuntu runner with 'the call had not returned after 30s' for a config-only change (PR #859, run 37710101909); it passed on rerun, so its fixed wall-clock wait is flaky under load.
