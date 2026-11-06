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

The status bar's context gauge and the session's real distance from compaction disagree once a person lowers the host's auto-compact window. With that window set well below the model's full window, the host said 1% was left until auto-compact while the ctx gauge showed one dot of four. The bar reads context_window.used_percentage, and the reading fits that figure being measured against the model's whole window rather than the lowered one, though the payload itself was not captured to prove it. The result is that the gauge never nears full and its warning at 80% can never fire: compaction arrives with no notice. abcd's own renderer (internal/core/statusline/render.go, from ContextPct) takes the same field, so it shares the fault.
