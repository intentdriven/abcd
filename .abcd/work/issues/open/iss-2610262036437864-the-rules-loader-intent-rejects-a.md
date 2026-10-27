---
schema_version: 1
id: "iss-2610262036437864"
slug: "the-rules-loader-intent-rejects-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 669 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-3]
finding_ref: cfk-f7d51eed8e4b8eee
---

The rules-loader intent rejects a user-level rules.json and keeps rules per repository; the guard chapter counts a rules.json in abcd's home folder as part of the user layer.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-3-modular-rules-loader.md:41` — "explicitly rejected. Keeping a personal cross-repo rules file recreates exactly the"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:452` — "its trusted-roots, rules.json, config.json and the rest of the user layer"

itd-3 marks a global rules.json in abcd's home as explicitly rejected and keeps rules per repository only, because a cross-repository rules file recreates the failure the loader exists to fix. The guard chapter names rules.json, beside trusted-roots and config.json, among the files of abcd's own folder in the home directory, part of the user layer. One document says the file must not exist and the other treats it as part of abcd's home.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 669.
