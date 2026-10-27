---
schema_version: 1
id: "iss-2610262040034442"
slug: "the-readme-requires-a-flat-commands"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 906 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:352"
remedy: "none (filed automatically)"
related_intents: [itd-46]
finding_ref: cfk-0e1fc9a21c45b69c
---

The README requires a flat commands/ directory where a plugin-named subdirectory breaks every verb; itd-46 edits the intent page at commands/abcd/intent.md

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:352` — "verb one level down in a subdirectory named after the plugin registers as `/abcd:abcd:<verb>`"
- End B: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:50` — "Update `abcd-cli/commands/abcd/intent.md`"

The README says commands/ is flat on purpose, because a verb in a subdirectory named after the plugin registers as /abcd:abcd:< verb>, which makes every documented /abcd:< verb> an unknown command. itd-46 puts the intent command page at abcd-cli/commands/abcd/intent.md, which is that subdirectory. The page cannot sit there if the directory is to stay flat.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 906.
