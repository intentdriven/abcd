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
deferred_after: "v0.11.1"
deferral_reason: "fixable, queued for a code lane (rulings-owed G; lane reanchor, run A 2026-09-29): the fork this record asks is answered in code for the directory half, since the transcript store creates every level 0o700 because 'the store is private' (internal/core/history/location.go, storeDirPerm) and ahoy no longer lays the chain out; each record is still written 0o644 (internal/core/history/history.go, Capture), and aligning it to 0o600 is a mechanical change with a test on the resulting mode"
---

Transcript records are written world-readable: history.Capture writes each redacted record with mode 0644 into a transcripts chain ahoy creates 0755 (apply.go), while the unredacted staging tier is 0700/0600 by stated design. Noted by GHSA-gmp7-9rvm-qcr3 as the amplifier of the PEM leak, it is a separate question from the redaction defect: whether a redacted transcript store under the caller home should be private to the account (records 0600 and the chain 0700) or is meant to be readable by other local accounts. No ADR or decision records the choice, the tightening spans the ahoy-owned directory chain, and no consumer is known to rely on a world-readable store. Fork to decide: tighten both (one mode change in history plus one in ahoy, with a test on the resulting mode) or record the world-readable posture as deliberate.
