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
resolution: "mode.CanSet probes that abcd mode could write the state (real tier, nothing unreplaceable at the store's path, a file creatable and removed in the tier); the question gate asks it before refusing and fails open loud (exit 1, NOT CHECKED) where the verb could not answer the refusal. HasTier's comment says it tests presence only."
impact: internal
resolved_by:
  commit: "ce212df6"
---

The question gate can refuse a legitimate question forever: in a managed checkout whose .abcd/.work.local/ exists but is not writable, HasTier reads true and the absent mode file reads managed, so the guard exits 2 on the host's question tool, while the remedy it names, abcd mode facilitator, fails with permission denied. HasTier tests presence, not writability, though its comment says it gates on the verb being able to set the state.

## Grounds

- pursued: a question in a managed checkout with a read-only tier runs on exit 1 with a NOT CHECKED line instead of being refused forever; shown wrong if TestGuardQuestionGateFailsOpenWhereTheModeCannotBeSet sees exit 2 or residue in the tier
