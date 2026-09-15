---
schema_version: 1
id: "iss-2608300349499636"
slug: "outstanding-report-silent-on-contested-items"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "itd-180 fourth-round security review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go (standing[0] selection)"
resolution: "The report names every standing answer as a Contested fault instead of picking the first by id, and every standing hold publishes its exit condition whatever else is true of the item; the parity table now asserts the full standing set on both counts."
impact: fix
---

Two independent standing dispositions on one item — the ordinary result of two branches each answering the item and merging without conflict — make the outstanding report take the first by id and say nothing: an accepted record first hides a held record and its exit condition, and no line says two answers stand, contradicting the rule's own contract that silence is the one answer it must never give by accident; the parity test builds its expectation from the first id so it cannot notice. Emit a contested fault naming every standing id when more than one stands.
