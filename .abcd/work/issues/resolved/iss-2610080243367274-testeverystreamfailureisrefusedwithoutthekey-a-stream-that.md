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
resolution: "The never-ending-stream case of TestEveryStreamFailureIsRefusedWithoutTheKey now sets a short total cap on the client (WithTimeout) and waits capWait(cap) = 3x the cap instead of a fixed 30s, accepting the cap's refusal beside the byte bound's; a second case streams the flood too slowly to reach the bound, reproducing the CI failure under the old wait and refused by the cap in 2s after."
impact: internal
resolved_by:
  commit: "f69a86ab9362d5659983d3080e254bac89b5ab40"
---

TestEveryStreamFailureIsRefusedWithoutTheKey/a_stream_that_never_ends failed on the CI ubuntu runner with 'the call had not returned after 30s' for a config-only change (PR #859, run 37710101909); it passed on rerun, so its fixed wall-clock wait is flaky under load.

## Grounds

- pursued: the case returns within a wait derived from the adapter's own deadline whatever the runner's throughput; a CI run failing it with 'the call had not returned after' a wait derived from the cap would show it wrong
