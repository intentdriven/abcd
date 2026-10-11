---
schema_version: 1
id: "iss-2609090951276167"
slug: "attribution-machine-signal-is-bot-suffix"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-attribution.sh"
resolution: "The attribution gate also refuses, in both roles, a trailing bot, robot or automation word ending the display name or the mailbox's local part, standing alone or joined by - or _ (and + in the local part), so a configured automation such as semantic-release-bot at a forge no-reply address is a machine; whitespace does not separate the word in a display name, so a person named Jan Bot passes (cec6047b5), and the comment names the shapes still out of reach. Proved by cases in scripts/check-attribution-cases.sh: semantic-release-bot at a forge no-reply address and by name alone, ci_bot, Renovate Bot at bot@renovateapp.com, release_automation@ and a ci-robot committer are refused, while Ada Talbot at a forge privacy address, jean.bot@ and Jan Bot pass."
impact: internal
resolved_by:
  commit: "bd3f55e68"
---

The attribution gate states its rule as refuse machines and allow humans, and implements the machine half structurally rather than nominally: a name ending in the forge-stamped bot suffix, a mailbox carrying that suffix, or one vendor domain. Its comment claims a second automation lands in the right place with no edit here, which holds only for automations the forge itself stamps. An identity such as semantic-release-bot with a forge no-reply address matches neither the AI name list nor the AI mail list, neither machine pattern, and not the author-only no-reply rule, so it is judged a human and the gate exits clean; verified by reading the four patterns against that identity. Self-hosted release automation, CI bots committing under a plain configured name, and any forge whose suffix is not the one hard-coded here all land the same way, and this gate is the only thing standing between them and the contributor graph the rule exists to protect. It matters because the rule was written after a bot walked past the nominal list, and the structural replacement inherits the same enumeration in a different alphabet. Fix direction: widen the structural signal beyond one forge suffix, whether by treating a trailing bot or automation token in the name or local part as machine-shaped, by keeping an automation mailbox list beside it, or by requiring a positive human signal, and say in the comment which shapes remain out of reach. Detector: a commit authored as semantic-release-bot with a forge no-reply address must be refused as a machine, while an outside human contributor with a forge privacy address still passes.

## Grounds

- pursued: a self-configured automation identity is refused while outside humans at forge privacy addresses still pass; a full-history run refusing a human commit the previous gate passed would show it wrong
