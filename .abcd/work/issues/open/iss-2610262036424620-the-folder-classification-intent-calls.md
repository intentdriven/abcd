---
schema_version: 1
id: "iss-2610262036424620"
slug: "the-folder-classification-intent-calls"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 668 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-40-folder-classification.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-40]
finding_ref: cfk-70ab1952c588673f
---

The folder-classification intent calls ahoy's registry of managed repositories "the history store"; in the history chapter that store is the transcript store, kept apart from the registry.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-40-folder-classification.md:16` — "the history store's `index.json` keyed on each repo's root-commit SHA"
- End B: `.abcd/development/brief/04-surfaces/11-history.md:15` — "ahoy's registry stays under `~/.abcd.noindex/history/` and holds no transcripts."

itd-40 names the machine-wide registry of managed repositories the history store's `index.json`, under a scope heading "The history-store registry". The history chapter makes the `history` verb's store the session-transcript store under `transcripts/<root-sha>/records/`. It says ahoy's registry under `history/` holds no transcripts, and that install provisions the registry, not the store. The one name "history store" therefore stands for two different stores.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 668.
