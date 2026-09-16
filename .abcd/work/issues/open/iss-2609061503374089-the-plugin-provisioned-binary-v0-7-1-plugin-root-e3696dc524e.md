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
---

The plugin-provisioned binary (v0.7.1, plugin root e3696dc524e3) has no decide verb, but the plugin ships the /abcd:decide skill, which tells a plugin user to run '<plugin-root>/abcd decide' and reports an unknown-command refusal. Observed on 2026-09-06 in a managed repo when minting its first ADR; the source-checkout build has the verb. Either the plugin payload lags the skill it documents, or the skill should state the minimum binary version and the ahoy status should report the gap. Same session also found the intent skill referring to .abcd/config/identity.json and to research/notes/, neither of which ahoy install scaffolds in a fresh managed repo.
