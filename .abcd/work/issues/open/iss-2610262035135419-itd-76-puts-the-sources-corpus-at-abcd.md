---
schema_version: 1
id: "iss-2610262035135419"
slug: "itd-76-puts-the-sources-corpus-at-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 556 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-76]
finding_ref: cfk-5d164db05a8635b8
---

itd-76 puts the sources corpus at ~/.abcd/sources/ by default; the library chapter puts it at ~/.abcd.noindex/sources

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-76-source-provenance-ledger.md:18` — "abcd's user-level home, `~/.abcd/sources/` by default, itself a no-remote git repository"
- End B: `.abcd/development/brief/04-surfaces/07-library.md:64` — "The corpus is the default one under the user-level home, `~/.abcd.noindex/sources`, unless the invocation names another directory."

Both documents name the default location of the same user-level sources corpus. itd-76 gives ~/.abcd/sources/ and the library chapter gives ~/.abcd.noindex/sources. Only one can be the default, so a person or tool following the other will look in the wrong directory.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 556.
