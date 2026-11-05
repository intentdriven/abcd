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
---

A drain's order is fixed: category, then severity, then its tie-breaks (orderEligible in internal/core/capture/eligible.go), and its flags can only hold an issue back (--skip), so there is no way to name issues to take first. A bug that a person or a release cut marks urgent waits behind the whole ledger.
