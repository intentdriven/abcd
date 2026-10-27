---
schema_version: 1
id: "iss-2610262039345320"
slug: "itd-39-has-the-prompt-hook-inject-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 869 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-39]
finding_ref: cfk-a921667857cba965
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The surfaces README table is a statement of what ships; itd-39 is planned and its spec (acceptance N4, step 7) updates the memory-beside-library wording when the bundle ships."
---

itd-39 has the prompt hook inject `.abcd/memory/` pages automatically on a prompt match; the surfaces register says library material comes back only when the person asks

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-39-scope-aware-memory-retrieval.md:20` — "also scans each prompt against memory pages across two scopes"
- End B: `.abcd/development/brief/04-surfaces/README.md:491` — "The person asks ("what do my sources say about pacing?") and gets answers that cite the item they came from"

itd-39 has the UserPromptSubmit hook scan each prompt against memory pages and inject the matching ones, and its decisions record that the store holding those pages is the library's (`library ingest`, `library ask`, `library lint`). The surfaces register's 'two different jobs' table says library items come back when the person asks and get answers that cite their source. Only memory notes are brought into a session automatically, and it says the two are not one store under two names. Auto-injecting library pages erases the split the register describes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 869.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The surfaces README table is a statement of what ships; itd-39 is planned and its spec (acceptance N4, step 7) updates the memory-beside-library wording when the bundle ships.
