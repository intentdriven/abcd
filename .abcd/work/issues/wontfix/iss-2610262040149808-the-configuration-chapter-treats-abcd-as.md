---
schema_version: 1
id: "iss-2610262040149808"
slug: "the-configuration-chapter-treats-abcd-as"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 918 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:702"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-8916671d753125bd
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The rename intent's decision 2 moves the whole home and lists sources among its stores; itd-76's old spelling was true when written."
---

The configuration chapter treats ~/.abcd as the retired home, read by no reader; itd-76 puts the sources corpus at ~/.abcd/sources/ by default

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:702` — "The folder's earlier name, `~/.abcd`, is read by `abcdhome.Check` alone, with `Lstat`, and never by a reader, so no reader falls back to it."
- End B: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default"

03-configuration says `~/.abcd` is the user scope's earlier name, read only by abcdhome.Check and never by a reader. While anything stands there, every command and hook writes nothing and reports the stop, and the sources/ corpus belongs under ~/.abcd.noindex/. itd-76 gives `~/.abcd/sources/` as the corpus's default location. Creating the corpus where itd-76 says would halt every abcd command.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 918.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The rename intent's decision 2 moves the whole home and lists sources among its stores; itd-76's old spelling was true when written.
