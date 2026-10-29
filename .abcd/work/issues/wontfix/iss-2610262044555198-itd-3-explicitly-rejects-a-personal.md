---
schema_version: 1
id: "iss-2610262044555198"
slug: "itd-3-explicitly-rejects-a-personal"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1239 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117, itd-3]
finding_ref: cfk-99536d0b9caf9e74
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-3's delivered-versus-promised notes name itd-117 and cite internal/core/rules/rules.go for the user layer; nothing needs an edit."
---

itd-3 explicitly rejects a personal cross-repo ~/.abcd/rules.json; itd-117 ships exactly that user-scope rules file

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"
- End B: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41` — "explicitly rejected. Keeping a personal cross-repo rules file recreates exactly the"

itd-3 records a global ~/.abcd/rules.json as explicitly rejected and rules the loader per-repo only, because a personal cross-repo rules file recreates the scaffolding-outside-the-repo failure. itd-117 introduces a user-scope rules layer in that same file, injected into every repository abcd manages, and does not record that it reverses itd-3. The two records cannot both describe the loader's layers: one says the user layer is rejected by design, the other says it exists between the bundled defaults and the repo file.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1239.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. itd-3's delivered-versus-promised notes name itd-117 and cite internal/core/rules/rules.go for the user layer; nothing needs an edit.
