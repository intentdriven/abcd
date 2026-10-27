---
schema_version: 1
id: "iss-2610262044322326"
slug: "the-worktree-block-intent-assigns-typed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1213 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-148-every-change-starts-in-its-own-worktree.md:101"
remedy: "none (filed automatically)"
related_intents: [itd-148, itd-2609091034175565]
finding_ref: cfk-7c49c784bda5c8b4
---

The worktree-block intent assigns typed claims to itd-33, but the shipped reservation intent delivered the claim

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-148-every-change-starts-in-its-own-worktree.md:101` — "The coordination mechanism — typed claims, take or yield or escalate — remains itd-33's"
- End B: `.abcd/development/intents/shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md:239` — "the record keeps "claim" as the internal name"

itd-148 says the coordination mechanism of typed claims remains itd-33's and puts claims out of scope as itd-33's. itd-2609091034175565 shipped exactly that claim, under the person-facing name 'reservation' with 'claim' kept as the internal name. Ownership of the claim is assigned to two records.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1213.
