---
schema_version: 1
id: "iss-2611061848556595"
slug: "ctx-gauge-ignores-a-lowered-compact"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "a long session after lowering the auto-compact window"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/statusline/payload.go"
remedy: "Measure the gauge against the lower of the model's window and the auto-compact window the host declares for that model, reading the token count the payload carries; when the setting cannot be read, say the reading is against the full window, and capture a real payload in a test fixture first to confirm the cause."
---

The status bar's context gauge and the session's real distance from compaction disagree once a person lowers the host's auto-compact window. With that window set well below the model's full window, the host said 1% was left until auto-compact while a context gauge showed one dot of four. That gauge was a status-line script outside this tree, reading context_window.used_percentage; the reading fits that figure being measured against the model's whole window rather than the lowered one, though the payload itself was not captured to prove it. abcd's own row (internal/core/statusline/render.go) prints a plain ctx percentage from ContextPct, which takes the same field, so it would share the fault and read low as compaction nears. ContextPct also feeds the fill file behind the rules loader's brackets, and the transcript fallback in fill.go divides by a per-model table, so a fix has to reach both.
