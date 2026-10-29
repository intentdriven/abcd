---
schema_version: 1
id: "iss-2610262045266820"
slug: "itd-3-rejects-a-user-level-rules-json"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1274 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-3, itd-2610030720038073]
finding_ref: cfk-824e49b65fb5e817
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-3's rejection was true in its day; itd-117 reversed it and the loader now composes bundled, user and repo layers (internal/core/rules/layerfile.go), as the later home-folder record also states."
---

itd-3 rejects a user-level rules.json outright; itd-2610030720038073 records that every managed project's block names the home's top-level rules.json

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41` — "**Global `~/.abcd/rules.json`** — explicitly rejected."
- End B: `.abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md:95` — "each managed project's block names the home's top-level `rules.json`"

itd-3 lists a global ~/.abcd/rules.json as explicitly rejected and the override as per-repo only. itd-2610030720038073, answering from the code, says each managed project's block names the home's top-level rules.json, which is that same personal cross-repo layer. The loader either has a machine layer or it does not; the two shipped records say opposite things and neither marks the other as superseded.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1274.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-3's rejection was true in its day; itd-117 reversed it and the loader now composes bundled, user and repo layers (internal/core/rules/layerfile.go), as the later home-folder record also states.
