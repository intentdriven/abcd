---
schema_version: 1
id: "iss-2610111730597708"
slug: "guard-unset-cdpath-still-blocks"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Track unset of CDPATH (and cdpath) on the line so later cd operands are judged without it; prove with a Check test that unset CDPATH; cd name allows while CDPATH=x cd name still blocks"
---

After `unset CDPATH` on the line, the guard falls back to the hook environment's CDPATH, so a `cd name` that bash would resolve without CDPATH is still blocked: a false block for a person who exports CDPATH, the one group the new CDPATH block reaches. Left open by lane G on 2026-10-11 (iss-2610111600031244).
