---
schema_version: 1
id: "iss-2610262040508345"
slug: "itd-37-s-discipline-says-every-spec"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 959 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:17"
remedy: "State in itd-37's delivered-state paragraph that the section is not yet required in practice: no committed spec carries it, abcd spec scaffolds none and no lint checks it (verification matrix row is staged)"
related_intents: [itd-37]
finding_ref: cfk-aedbbff530d3f178
---

itd-37's discipline says every spec carries a Modification Grammar section; the verification matrix says no committed spec carries one and nothing checks for it.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:17` — "Every spec carries a `## Modification Grammar` section with three sub-headings"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:94` — "No committed spec carries the section, `abcd spec` scaffolds none, the codes exist in no lint, and nothing writes the pages"

itd-37 is a discipline in force: every spec carries a ## Modification Grammar section, and every spec plan-reviewed under abcd inherits its gates. The matrix says the section is staged in full. No committed spec carries it, abcd spec scaffolds none, the MG codes exist in no lint, and nothing writes the memory pages. The rule and the record of what was delivered cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 959.
