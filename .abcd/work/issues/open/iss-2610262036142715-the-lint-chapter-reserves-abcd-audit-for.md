---
schema_version: 1
id: "iss-2610262036142715"
slug: "the-lint-chapter-reserves-abcd-audit-for"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 633 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/16-lint.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-85]
finding_ref: cfk-9e702702904fde00
---

The lint chapter reserves /abcd:audit for itd-16's hash-chain fidelity surface and calls the conformance check abcd lint; itd-85, the intent the chapter cites, ships that same check as abcd audit.

A naming conflict (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/16-lint.md:20` — "`/abcd:audit` stays reserved for itd-16's hash-chain fidelity surface."
- End B: `.abcd/development/intents/drafts/itd-85-audit-verb.md:16` — "**abcd ships `abcd audit`** — a read-only check that tells you, in one command, whether a repository follows the working conventions"

16-lint says the conformance verb applies rules about form and so is a lint, and that /abcd:audit stays reserved for itd-16's hash-chain fidelity surface, where the staged chain and lifeboat verbs belong. itd-85 announces the same read-only repo-conformance check, with the same five v1 rules, as a new abcd audit verb with its own commands/audit.md. So one name, audit, is given to two different surfaces, and building itd-85 as written would collide with the reserved one.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 633.
