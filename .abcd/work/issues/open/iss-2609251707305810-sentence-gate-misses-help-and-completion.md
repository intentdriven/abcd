---
schema_version: 1
id: "iss-2609251707305810"
slug: "sentence-gate-misses-help-and-completion"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/sentences_test.go"
---

The sentence gate walks the command tree before cobra adds help and completion at Execute, so abcd --help lists two framework verbs with cobra's own text that the gate never sees and no test names as exempt: the check reads as covering every listed verb while two are silently outside it
