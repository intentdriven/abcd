---
schema_version: 1
id: "iss-2609090951276167"
slug: "attribution-machine-signal-is-bot-suffix-shaped-only"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-attribution.sh"
---

The attribution gate states its rule as refuse machines and allow humans, and implements the machine half structurally rather than nominally: a name ending in the forge-stamped bot suffix, a mailbox carrying that suffix, or one vendor domain. Its comment claims a second automation lands in the right place with no edit here, which holds only for automations the forge itself stamps. An identity such as semantic-release-bot with a forge no-reply address matches neither the AI name list nor the AI mail list, neither machine pattern, and not the author-only no-reply rule, so it is judged a human and the gate exits clean; verified by reading the four patterns against that identity. Self-hosted release automation, CI bots committing under a plain configured name, and any forge whose suffix is not the one hard-coded here all land the same way, and this gate is the only thing standing between them and the contributor graph the rule exists to protect. It matters because the rule was written after a bot walked past the nominal list, and the structural replacement inherits the same enumeration in a different alphabet. Fix direction: widen the structural signal beyond one forge suffix, whether by treating a trailing bot or automation token in the name or local part as machine-shaped, by keeping an automation mailbox list beside it, or by requiring a positive human signal, and say in the comment which shapes remain out of reach. Detector: a commit authored as semantic-release-bot with a forge no-reply address must be refused as a machine, while an outside human contributor with a forge privacy address still passes.
