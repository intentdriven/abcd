---
schema_version: 1
id: "iss-2610262044513201"
slug: "shipped-itd-48-says-role-2-findings-are"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1235 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:121"
remedy: "none (filed automatically)"
related_intents: [itd-48, itd-2609291923559186]
finding_ref: cfk-21e9e4350776a222
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. An out-of-scope line limits the one intent, not later ones; itd-2609291923559186 builds_on itd-48, is still planned, and makes automatic a setup choice rather than the default."
---

Shipped itd-48 says Role 2 findings are only reported and a person acts on them; itd-2609291923559186 adds an automatic mode that fixes mechanical contradictions in the commit

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:121` — "Auto-fix for findings.** Role 2 and Role 3 surface findings; a"
- End B: `.abcd/development/intents/planned/itd-2609291923559186-the-consistency-pass-runs-where-records.md:150` — "warn, stop, or automatic (fix the obvious, file the rest)"

itd-48 puts auto-fix for findings out of scope: Role 2 and Role 3 surface findings, and a person, or `reclassify`, acts on them. itd-2609291923559186 makes its A1 codes the mechanical half of Role 2. It also lets a repository choose an automatic mode at setup, which fixes the obvious contradictions in the same commit and files the rest. With that setting, the tool applies a Role 2 finding with no person acting, which itd-48 says does not happen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1235.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. An out-of-scope line limits the one intent, not later ones; itd-2609291923559186 builds_on itd-48, is still planned, and makes automatic a setup choice rather than the default.
