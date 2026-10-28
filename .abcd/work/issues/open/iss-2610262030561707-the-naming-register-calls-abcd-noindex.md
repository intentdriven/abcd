---
schema_version: 1
id: "iss-2610262030561707"
slug: "the-naming-register-calls-abcd-noindex"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 64 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:227"
remedy: "none (filed automatically)"
finding_ref: cfk-8cc23fd3a42b01e6
---

The naming register calls `~/.abcd.noindex/history/` the history store's registry; the history chapter says that directory is ahoy's registry, holds no transcripts, and that the history store lives under `~/.abcd.noindex/transcripts/`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:227` — "History-store registry at `~/.abcd.noindex/history/index.json` recording each repo's identity + lineage."
- End B: `.abcd/development/brief/04-surfaces/11-history.md:15` — "ahoy's registry stays under `~/.abcd.noindex/history/` and holds no transcripts."

04-naming registers `index.json` as the 'History-store registry at `~/.abcd.noindex/history/index.json`', and its voyage row keys records 'like the history store', so in that page 'history store' names the `history/` directory. 11-history defines the store the `history` verb serves as `~/.abcd.noindex/transcripts/<root-sha>/records/`, and says ahoy's registry stays under `~/.abcd.noindex/history/` and holds no transcripts. The one name 'history store' points at two different directories with different contents.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 64.
