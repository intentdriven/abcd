---
schema_version: 1
id: "iss-2610262040588956"
slug: "a-person-s-preferences-have-two"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 968 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:14"
remedy: "none (filed automatically)"
related_intents: [itd-2610091918433290]
finding_ref: cfk-34ef30ef092f5f20
---

A person's preferences have two user-level homes: the memory chapter's `~/.abcd.noindex/memory/` pages and the memory-notes intent's `~/.abcd.noindex/memory.json`.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:14` — "`~/.abcd.noindex/memory/` | **Personal preferences** and cross-project principles that have no single repo home"
- End B: `.abcd/development/intents/planned/itd-2610091918433290-an-agent-working-in-an-abcd-managed.md:64` — "The machine's `~/.abcd.noindex/memory.json` holds personal notes"

07-memory.md sends personal preferences and cross-project lessons to the user-scope page store `~/.abcd.noindex/memory/`, curated by principle-distiller. itd-2610091918433290 puts personal notes, which the glossary describes as a person's preferences and lessons, in `~/.abcd.noindex/memory.json`, written by agents. Neither says which store a personal preference belongs in, so the same knowledge is claimed by two stores with different writers and lifecycles.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 968.
