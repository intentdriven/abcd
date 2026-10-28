---
schema_version: 1
id: "iss-2610262037246775"
slug: "itd-37-has-an-agent-principle-distiller"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 720 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:63"
remedy: "none (filed automatically)"
related_intents: [itd-37]
finding_ref: cfk-58e0c1b06c34a57a
---

itd-37 has an agent (principle-distiller) write memory pages into the repository's .abcd/memory/; 37-memory says agents write only the machine layer.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:63` — "Writes append-only `spec_modification_grammar_<spec_id>.md` to `.abcd/memory/`"
- End B: `.abcd/development/brief/04-surfaces/37-memory.md:50` — "Agents write only the machine layer, and only through one gate"

37-memory lets agents write only the machine file, through one gate; the repository's memory changes only by a reviewed edit, and there is no promote operation. itd-37 has the principle-distiller agent write append-only and curated pages into the repository's .abcd/memory/ at every spec completion. Either agents write repository memory or they do not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 720.
