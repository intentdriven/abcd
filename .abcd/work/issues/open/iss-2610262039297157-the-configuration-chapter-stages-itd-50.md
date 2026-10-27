---
schema_version: 1
id: "iss-2610262039297157"
slug: "the-configuration-chapter-stages-itd-50"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 864 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:98"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-4bc2a0e6ab6ec723
---

The configuration chapter stages itd-50's audit loop as a per-intent frontmatter mode, while itd-50 rules that the loop is build's last stage for every intent, not a per-intent mode

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:98` — "not a per-intent mode; every intent the run ships goes through it."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:417` — "one intent can loop while another stays record-only"

The configuration chapter says the audit-loop mode and budget are elected per intent in the intent's own frontmatter (`audit_mode`, `audit_budget` default 3), so that one intent can loop while another stays record-only. itd-50's decisions of 2026-09-21 rule that the loop is not a per-intent mode: every intent a build run ships goes through it, and the round count is a per-run value (`--fix-rounds`, `pace.fix_rounds`, bundled 3). A builder following the brief would add per-intent frontmatter keys and a per-intent opt-in that the intent's own rulings replaced.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 864.
