---
schema_version: 1
id: "iss-48"
slug: "e2e-behavioural-scenarios"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "evals/smoke_test.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan a behavioural end-to-end scenario suite per verb family against the built binary, beyond M11's write-path smoke?"
---

behavioural end-to-end coverage: evals/smoke_test.go is structural only (help renders, no panic) — no scenario drives the built binary through a real verb round-trip (capture an issue, resolve it, list it; ahoy install into a temp repo; memory ask against a fixture substrate). Detector: a small scenario suite against the cross-compiled binary, one scenario per verb family, run in CI on both platforms. Acceptance corpus: the verb families with zero behavioural e2e today — capture, ahoy, memory, docs, history, launch.