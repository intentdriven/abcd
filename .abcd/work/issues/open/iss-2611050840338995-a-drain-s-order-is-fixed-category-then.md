---
schema_version: 1
id: "iss-2611050840338995"
slug: "a-drain-s-order-is-fixed-category-then"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "coordinating build lanes, 2026-11-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/capture/eligible.go"
remedy: "Give the drain a way to name issues for the front of its order, each still judged by the drain rule and named as moved in the dry run and the run's summary. A test names a later issue and asserts its lane opens first."
related_issues: [iss-2610161655393137]
---

A running drain's own order is fixed: category, then severity, then its tie-breaks (orderEligible in internal/core/capture/eligible.go), and its flags can only hold an issue back (--skip), so it cannot be told which eligible issue to take first. The route a person has today is to build one issue themselves with `abcd build <iss-N>` (internal/surface/cli/build.go:102), under the same drain rule. Inside a running drain, an issue a person or a release cut wants sooner waits behind every eligible issue ahead of it.

This concerns only the issues in that order: a major is not in a baseline drain's order at all, and the ledger holds no mark of urgency a drain could read. iss-2610161655393137 carries a ruling that one repository's drain runs most urgent first; that orders by severity and names no issue, so this record neither duplicates it nor is a narrower case of it.
