---
schema_version: 1
id: "iss-2610262033586792"
slug: "itd-5-relies-on-intent-auditor-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 459 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:102"
remedy: "none (filed automatically)"
related_intents: [itd-5]
finding_ref: cfk-dcc9823dbb4870a1
---

itd-5 relies on intent-auditor's single-document role checking delivery against itd-5's own criteria; the audit refuses any record in disciplines/

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-5-prompt-quality-additions.md:102` — "checks delivered reality against this discipline's acceptance criteria when each agent spec ships"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:326` — "Takes a **shipped** intent and nothing else: a record still in `drafts/`, `planned/`, `disciplines/` or `superseded/` is refused by name"

itd-5, a discipline, says the auditor's single-document role checks delivered reality against this discipline's acceptance criteria when each agent spec ships. The intent surface's audit takes a shipped intent and nothing else and refuses a record in disciplines/ by name. No audit run can judge a discipline's criteria, so the check itd-5 depends on cannot be issued by the surface as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 459.
