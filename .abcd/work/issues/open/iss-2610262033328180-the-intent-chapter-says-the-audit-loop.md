---
schema_version: 1
id: "iss-2610262033328180"
slug: "the-intent-chapter-says-the-audit-loop"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 418 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:578"
remedy: "Rewrite the Audit-loop mode and budget paragraph at 03-configuration.md lines 415-427 per iss-2610160855131633 to describe the loop as abcd build's last stage (itd-50 decision 1), or drop the per-intent keys."
finding_ref: cfk-f94f53f1a6997d34
---

The intent chapter says the audit loop was re-scoped into the last stage of the build, while the configuration chapter still stages it as a per-intent mode.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:578` — "This per-intent shape predates itd-50's 2026-09-21 re-scope, which makes the loop the last stage of `abcd build` rather than a per-intent mode"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:415` — "**Audit-loop mode and budget (staged: itd-50)** are elected **per intent**, in the intent's own frontmatter rather than in `config.json`"

05-intent records that the per-intent audit-mode shape predates a re-scope that makes the loop the last stage of abcd build rather than a per-intent mode. 03-configuration presents the staged audit-loop mode and budget as chosen per intent, in each intent's own frontmatter. Anyone building the loop from the configuration chapter would build the shape the intent chapter calls superseded.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 418.
