---
schema_version: 1
id: "iss-2610121224332411"
slug: "ahoy-refused-answer-note-only-scan-deep"
severity: "minor"
category: "ux"
source: "impl-review"
found_during: "lane for iss-2610020700263496, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/apply.go"
remedy: "Give every value question ahoy install prompts (visibility, docs_target, oracle_backend) the refused-answer note scan_deep gets after iss-2610020700263496: name the answer refused and the flag that sets the value instead; one test per question that a refused answer prints its note, watched fail first."
refines: [iss-2610031236155833, iss-2609261447395216]
blocked_by: [iss-2610020700263496]
---

When ahoy install refuses a prompted answer to the visibility, docs_target or oracle_backend question (a piped 'y' where a choice is expected, say), it says nothing about the refused answer: the person sees only the gap that remains. The fix for iss-2610020700263496 adds a note naming the refused answer for scan_deep alone, as that remedy asks, and a retired docs target already has its own note; the other value questions stay silent. Found by the lane for iss-2610020700263496 (abcd-0d, 2026-10-12).
