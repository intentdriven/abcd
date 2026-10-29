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
remedy: "Rewrite the 'Audit-loop mode and budget (staged: itd-50)' paragraph to the re-scope (build's last stage, --fix-rounds and pace.fix_rounds, bundled 3) or drop the per-intent audit_mode and audit_budget keys; merge with the open iss-2610160855131633, which already carries this remedy."
related_intents: [itd-50]
finding_ref: cfk-4bc2a0e6ab6ec723
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The configuration chapter stages itd-50's audit loop as a per-intent frontmatter mode, while itd-50 rules that the loop is build's last stage for every intent, not a per-intent mode

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:98` — "not a per-intent mode; every intent the run ships goes through it."
- End B: `.abcd/development/brief/05-internals/03-configuration.md:417` — "one intent can loop while another stays record-only"

The configuration chapter says the audit-loop mode and budget are elected per intent in the intent's own frontmatter (`audit_mode`, `audit_budget` default 3), so that one intent can loop while another stays record-only. itd-50's decisions of 2026-09-21 rule that the loop is not a per-intent mode: every intent a build run ships goes through it, and the round count is a per-run value (`--fix-rounds`, `pace.fix_rounds`, bundled 3). A builder following the brief would add per-intent frontmatter keys and a per-intent opt-in that the intent's own rulings replaced.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 864.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
