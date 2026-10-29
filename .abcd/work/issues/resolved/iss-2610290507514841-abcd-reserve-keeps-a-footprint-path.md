---
schema_version: 1
id: "iss-2610290507514841"
slug: "abcd-reserve-keeps-a-footprint-path"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/claim/prepare.go"
remedy: "prepare drops a repeated footprint path, keeping the first, before the message is measured; the narrowing a renewal names is deduplicated the same way"
resolution: "prepare drops a repeated footprint path, keeping the first, before the message is measured; a renewal's narrowing is deduplicated the same way"
impact: fix
---

abcd reserve keeps a footprint path named twice: each --footprint is written as its own Abcd-Footprint line, so a renewal script that passes the held footprint again plus the new paths grows the message with repeats until it passes the 4096-byte cap and the renewal is refused.
