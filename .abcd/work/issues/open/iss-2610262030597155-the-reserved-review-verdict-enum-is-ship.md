---
schema_version: 1
id: "iss-2610262030597155"
slug: "the-reserved-review-verdict-enum-is-ship"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 70 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:187"
remedy: "none (filed automatically)"
finding_ref: cfk-dd270d0d74d01e65
---

The reserved review-verdict enum is SHIP/NEEDS_WORK/MAJOR_RETHINK, but the build loop parses SHIP/FIX FIRST and APPROVE/BLOCK/NEEDS-INPUT as review verdicts

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:187` — "Review verdicts ∈ `{SHIP, NEEDS_WORK, MAJOR_RETHINK}` | Carmack-style review verdicts"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:696` — "stating one verdict of its role (`SHIP` or `FIX FIRST`; `APPROVE`, `BLOCK` or `NEEDS-INPUT`)"

The naming register lists review verdicts as a controlled enum of exactly SHIP, NEEDS_WORK and MAJOR_RETHINK. The build chapter has the loop read each reviewer's one verdict as SHIP or FIX FIRST (and APPROVE, BLOCK or NEEDS-INPUT), tokens the register does not hold. Either the enum is not the closed list of review verdicts, or the loop parses verdicts outside it. A builder working from the register would reject the loop's tokens.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 70.
