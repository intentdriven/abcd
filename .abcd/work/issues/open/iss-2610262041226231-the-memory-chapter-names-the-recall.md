---
schema_version: 1
id: "iss-2610262041226231"
slug: "the-memory-chapter-names-the-recall"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 996 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/07-memory.md:197"
remedy: "Edit 07-memory.md section 9 (Diagnostic paragraph): replace the `recall [keyword]` sub-verb of `abcd library` with `abcd rules --prompt` on the rules verb, shipped per itd-39's 2026-10-11 decision."
related_intents: [itd-39]
finding_ref: cfk-87ed88b5278103c3
---

The memory chapter names the recall diagnostic a `recall` sub-verb of `abcd library`; itd-39 rules that the same diagnostic is `abcd rules --prompt` on the rules verb

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/07-memory.md:197` — "A `recall [keyword]` sub-verb of `abcd library` would show which pages a prompt or keyword surfaces"
- End B: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:78` — "is `abcd rules --prompt "<text>"`, on the existing `rules` verb"

Both documents describe one diagnostic: show which memory pages a prompt or keyword would surface, and in which bracket. The memory chapter places it as a `recall [keyword]` sub-verb of `abcd library`. itd-39's 2026-10-11 decision rules it is `abcd rules --prompt "<text>"` on the existing rules verb. One capability carries two different command names, so someone building from the brief would add a library sub-verb the planned intent has ruled out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 996.
