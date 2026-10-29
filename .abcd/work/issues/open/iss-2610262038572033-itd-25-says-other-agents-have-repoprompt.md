---
schema_version: 1
id: "iss-2610262038572033"
slug: "itd-25-says-other-agents-have-repoprompt"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 830 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:43"
remedy: "In itd-25 replace 'oracle backend chain (same as other agents)' with the shipped routing: an agent is host-delegated by default and routes per role through `roles.<role>.runner` or a provider connection (02-adapters.md, itd-2609201916056194, itd-2609170822093401); there is no Codex CLI adapter, and RepoPrompt routes only a build lane's two reviews."
related_intents: [itd-25]
finding_ref: cfk-2580fbf91e3bc973
---

itd-25 says other agents have RepoPrompt and Codex CLI alternatives; 02-adapters routes only a build lane's two reviews off the host

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-25-dredge-cross-corpus-synthesist.md:43` — "Oracle backend chain (same as other agents): host-delegated by default, with the opt-in RepoPrompt adapter and Codex CLI as configured alternatives"
- End B: `.abcd/development/brief/05-internals/02-adapters.md:191` — "A build lane's ruthless and security reviews, which the validate stage hands to the host's own agents by default, can be routed elsewhere two ways."

itd-25 gives its agent a backend chain 'same as other agents', with RepoPrompt and Codex CLI as configured alternatives to the host. 02-adapters says only a build lane's ruthless and security reviews can be routed elsewhere, through a claude or opencode runner or through RepoPrompt. No other agent has the chain itd-25 says is standard.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 830.
