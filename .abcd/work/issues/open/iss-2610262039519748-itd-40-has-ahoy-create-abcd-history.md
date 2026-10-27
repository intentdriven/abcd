---
schema_version: 1
id: "iss-2610262039519748"
slug: "itd-40-has-ahoy-create-abcd-history"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 891 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:702"
remedy: "none (filed automatically)"
related_intents: [itd-40]
finding_ref: cfk-8e286d9acb92b47c
---

itd-40 has ahoy create `~/.abcd/history/`; 03-configuration says no reader ever reads the old `~/.abcd` name and that its presence stops every command.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:702` — "The folder's earlier name, `~/.abcd`, is read by"
- End B: `.abcd/development/intents/shipped/itd-40-folder-classification.md:36` — "if `~/.abcd/history/` does not exist, the first `/abcd:ahoy install` creates it (directory + `index.json`) before registering the repo."

itd-40 says that if `~/.abcd/history/` does not exist, the first `/abcd:ahoy install` creates it. 03-configuration says the home is now `~/.abcd.noindex/`, that only the stop check reads the earlier name `~/.abcd`, and that anything standing there stops every command until it is renamed. Followed as written, itd-40 creates the very path the brief treats as a stop. Other shipped intents repeat the old spelling.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 891.
