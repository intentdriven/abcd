---
schema_version: 1
id: "iss-2610152246142642"
slug: "session-naming-s-prompt-hook-writes-a"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "final review of itd-2610111707085098, 2026-10-15"
origin: researcher-authored
production_mode: hand-written
found_at: "the session-naming prompt hook"
remedy: "Say a register refusal once per session (record it in the session's rules state, as the dedup ledger records injected domains) and stay silent on the prompts after it; test that a second prompt with the same refusal writes nothing to stderr."
---

Session naming's prompt hook writes a stderr note on every prompt of every titled session while the machine name register cannot be used: a sessions folder a group or others can write, one another account owns, or a register file that will not parse. Each note is accurate, but the same line repeats on every prompt until the person repairs the folder. Found by the final Opus review of itd-2610111707085098 (2026-10-15), recorded as not blocking.
