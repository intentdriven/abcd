---
schema_version: 1
id: "iss-2609261447395216"
slug: "ahoy-install-asks-private-repo-trufflehog-present-confirm"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (run A 2026-09-29, lane drainDrift3; rulings-owed BH): ahoy install asks to confirm deep secret scanning when a private repository has trufflehog on PATH and persists scan.deep, but nothing reads scan.deep and nothing runs trufflehog. Should abcd wire a trufflehog adapter behind the scanner seam so the answer takes effect, or stop asking and say the key is inert where it is already set?"
---

ahoy install asks 'Private repo + trufflehog present — confirm deep secret scanning' and persists scan.deep in .abcd/config.json, but no scanner reads scan.deep and nothing runs trufflehog, so the answer changes nothing: a person told they enabled deep secret scanning has not. Either wire a trufflehog adapter behind the scanner seam or stop asking (and say the value is inert where it is already set).
