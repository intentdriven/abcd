---
schema_version: 1
id: "iss-2610262033013099"
slug: "itd-85-wires-a-new-abcd-audit-verb-into"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 350 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:31"
remedy: "Supersede itd-85 by itd-124 (same act as the audit-name record); if it is kept instead, replace 'prepare-this-repo Phase 2' with the install section of commands/ahoy.md, whose conformance step already runs `abcd lint` in its JSON form."
related_intents: [itd-85]
finding_ref: cfk-dcd70b7d63c801da
---

itd-85 wires a new abcd audit verb into prepare-this-repo's Phase 2 as the conformance report; the ahoy page already gives that phase to abcd lint's JSON form.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:31` — "Wiring: `prepare-this-repo` Phase 2 consumes `abcd audit --json` in place of the hand-produced gap report."
- End B: `.abcd/development/brief/04-surfaces/01-ahoy.md:362` — "Run `abcd lint` in its JSON form, supplement it, write the gap report to the target's `.abcd/.work.local/scratch/`"

itd-85 makes `abcd audit` the read-only check of repository conformance (three-tier layout, conventions router, decision durability, docs currency) and has prepare-this-repo's Phase 2 consume `abcd audit --json` in place of the hand-produced gap report. The ahoy page says the preparation, now part of the install's plugin page, runs `abcd lint` in its JSON form at its conformance-lint phase, as the engine-backed conformance core, and writes the gap report from that. Two verbs claim the same conformance report for the same phase, and the page itd-85 wires into is retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 350.
