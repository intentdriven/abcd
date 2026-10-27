---
schema_version: 1
id: "iss-2610262045072539"
slug: "itd-117-puts-the-machine-s-shared"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1253 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-117, itd-2610030720038073]
finding_ref: cfk-1a439bfc01ce45c0
---

itd-117 puts the machine's shared conventions in ~/.abcd/rules.json; itd-2610030720038073 renames the home to ~/.abcd.noindex

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-117-development-instructions-live-in-the.md:19` — "now live once in `~/.abcd/rules.json` and inject into every"
- End B: `.abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md:95` — "rename the home itself, `~/.abcd` to `~/.abcd.noindex`"

itd-117 says the user-scope rules live once in ~/.abcd/rules.json. itd-2610030720038073 renames the home itself, ~/.abcd to ~/.abcd.noindex, which moves the home's top-level rules.json with it. The path itd-117 gives for the user rules layer is not where the renamed home keeps it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1253.
