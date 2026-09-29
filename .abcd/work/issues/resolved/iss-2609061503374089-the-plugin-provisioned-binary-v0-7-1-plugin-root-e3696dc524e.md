---
schema_version: 1
id: "iss-2609061503374089"
slug: "the-plugin-provisioned-binary-v0-7-1-plugin-root-e3696dc524e"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "2026-09-06 use in a managed repo"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/decide.md"
resolution: "commands/decide.md states the verb needs abcd 0.8.0 or later and routes an older binary's refusal to the release check in that binary's own spelling; the intent page says research/notes/ is not scaffolded, and ahoy install already writes identity.json. The record's second remedy, that the ahoy status should report the gap, is dropped rather than built: the payload lag that caused the gap is gone at 0.11.0, so the documented minimum version is the whole fix on the docs' merit"
impact: fix
resolved_by:
  commit: "07af2eb092552739a5360c39e5659457e7b3c734"
---

The plugin-provisioned binary (v0.7.1, plugin root e3696dc524e3) has no decide verb, but the plugin ships the /abcd:decide skill, which tells a plugin user to run '<plugin-root>/abcd decide' and reports an unknown-command refusal. Observed on 2026-09-06 in a managed repo when minting its first ADR; the source-checkout build has the verb. Either the plugin payload lags the skill it documents, or the skill should state the minimum binary version and the ahoy status should report the gap. Same session also found the intent skill referring to .abcd/config/identity.json and to research/notes/, neither of which ahoy install scaffolds in a fresh managed repo.

## Grounds

- pursued: a plugin user on an older binary is told why decide is unknown and how to update; a user still meeting a bare unknown-command with no route would show it wrong
