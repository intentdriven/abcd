---
schema_version: 1
id: "iss-2610262042125545"
slug: "the-agent-that-audits-intents-is-called"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1053 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-6fe4e9eb7c2681c9
---

The agent that audits intents is called `intent-fidelity-reviewer` in itd-48 and `intent-auditor` in the internals index.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:24` — "`intent-fidelity-reviewer` agent grows from one `role` to three"
- End B: `.abcd/development/brief/05-internals/README.md:29` — "When `intent-auditor` matures, brief-skeleton presence checks would land alongside its existing intent-vs-implementation checks."

itd-48 names the agent that holds Role 1 per-intent fidelity, and gains the cross-document and kind roles, `intent-fidelity-reviewer`. The internals index gives the name `intent-auditor` to the agent with the existing intent-vs-implementation checks. With two names for one agent, a reader cannot tell whether there is one prompt or two.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1053.
