---
schema_version: 1
id: "iss-2610262045296807"
slug: "itd-40-names-abcd-history-index-json-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1278 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-40-folder-classification.md:35"
remedy: "none (filed automatically)"
related_intents: [itd-40, itd-2610030720038073]
finding_ref: cfk-a562a6c70d4e05da
---

itd-40 names ~/.abcd/history/index.json as the sole user-scope registry; itd-2610030720038073 renames the home to ~/.abcd.noindex

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-40-folder-classification.md:35` — "`~/.abcd/history/index.json` is the sole user-scope registry"
- End B: `.abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md:21` — "The folder is named `~/.abcd.noindex`, a name the indexer passes over"

itd-40's classification keys a strong signal on the registry at ~/.abcd/history/index.json and bootstraps that directory on first install. itd-2610030720038073 moves abcd's whole home to ~/.abcd.noindex and refuses to run while ~/.abcd exists, so the registry path itd-40 states is no longer one abcd may read or create.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1278.
