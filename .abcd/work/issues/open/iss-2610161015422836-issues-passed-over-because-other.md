---
schema_version: 1
id: "iss-2610161015422836"
slug: "issues-passed-over-because-other"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Stop as empty only when nothing is eligible including held issues; otherwise wait and keep the drain's judgements"
related_intents: [itd-2610161016394967]
---

Issues passed over because other accounts hold them still let a drain stop as empty with complete true, so the next abcd drain begins a new drain with no stored judgements and judges everything again.
