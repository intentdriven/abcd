---
schema_version: 1
id: "iss-2610161015417715"
slug: "a-drain-s-pace-and-sub-agents-reach-only"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "multi-account drain investigation, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/drain.go"
remedy: "Carry the drain's resolved pace and sub-agents into every lane it opens"
related_intents: [itd-2610161016401115]
---

A drain's --pace and --sub-agents reach only the lane opened in its first move; later lanes resolve their pace again from configuration, so the old drain's first lane ran 10080/0 with six sub-agents and its next ran 120/300 with two.
