---
schema_version: 1
id: "iss-48"
slug: "e2e-behavioural-scenarios"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "2026-07-08 multi-agent review"
found_at: "evals/smoke_test.go"
related_intents: [itd-2609292109475690]
deferred_after: "v0.11.1"
deferral_reason: "The product thinker's ruling J24 of 2026-09-29: plan a behavioural end-to-end scenario suite per verb family. Filed as draft itd-2609292109475690. Owed: that draft's planning interview (the draft stays in drafts/ until a person plans it)."
---

behavioural end-to-end coverage: evals/smoke_test.go is structural only (help renders, no panic) — no scenario drives the built binary through a real verb round-trip (capture an issue, resolve it, list it; ahoy install into a temp repo; memory ask against a fixture substrate). Detector: a small scenario suite against the cross-compiled binary, one scenario per verb family, run in CI on both platforms. Acceptance corpus: the verb families with zero behavioural e2e today — capture, ahoy, memory, docs, history, launch.