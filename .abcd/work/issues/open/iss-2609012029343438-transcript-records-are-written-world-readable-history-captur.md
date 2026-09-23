---
schema_version: 1
id: "iss-2609012029343438"
slug: "transcript-records-are-written-world-readable-history-captur"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/history/history.go"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: tighten the transcript store to 0600/0700 or record the world-readable posture as deliberate (fork stated in the body)). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

Transcript records are written world-readable: history.Capture writes each redacted record with mode 0644 into a transcripts chain ahoy creates 0755 (apply.go), while the unredacted staging tier is 0700/0600 by stated design. Noted by GHSA-gmp7-9rvm-qcr3 as the amplifier of the PEM leak, it is a separate question from the redaction defect: whether a redacted transcript store under the caller home should be private to the account (records 0600 and the chain 0700) or is meant to be readable by other local accounts. No ADR or decision records the choice, the tightening spans the ahoy-owned directory chain, and no consumer is known to rely on a world-readable store. Fork to decide: tighten both (one mode change in history plus one in ahoy, with a test on the resulting mode) or record the world-readable posture as deliberate.
