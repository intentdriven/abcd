---
schema_version: 1
id: "iss-2609260100382261"
slug: "the-question-gate-can-refuse-a-legitimate-question-forever"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question.go"
---

The question gate can refuse a legitimate question forever: in a managed checkout whose .abcd/.work.local/ exists but is not writable, HasTier reads true and the absent mode file reads managed, so the guard exits 2 on the host's question tool, while the remedy it names, abcd mode facilitator, fails with permission denied. HasTier tests presence, not writability, though its comment says it gates on the verb being able to set the state.
