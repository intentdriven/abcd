---
schema_version: 1
id: "iss-2609230733586740"
slug: "itd-73-ac-5-promises-that-an-intent-captured-or-shaped-with"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
---

itd-73 ac-5 promises that an intent captured or shaped with an absent or invalid impact is flagged as a blocker; delivered reality: intent_impact_valid flags an invalid or internal value in every bucket but an ABSENT impact only in shipped/ (internal/core/lint/lint.go checkIntentImpact), and CreateFromText leaves impact optional on a draft — so a draft or planned intent carries no impact until the spec close forces one. Narrower than the criterion; spc-10 states the shipped/ bar only. Found by the itd-73 fidelity audit (rcp-0d18ea2d4682).
