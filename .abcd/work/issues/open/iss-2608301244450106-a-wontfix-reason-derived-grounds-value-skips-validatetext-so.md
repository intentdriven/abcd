---
schema_version: 1
id: "iss-2608301244450106"
slug: "a-wontfix-reason-derived-grounds-value-skips-validatetext-so"
severity: "nitpick"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-3-builder"
found_at: "internal/core/capture/grounds.go"
---

a wontfix reason-derived grounds value skips ValidateText so a control character in wontfix_reason still reaches yamlScalar

Surfaced by the itd-179 round-3 builder as an observation and captured rather
than fixed, because it is pre-existing and outside the round's records.

A wontfix whose grounds are DERIVED from the reason bypasses
`grounds.ValidateText`, so the control-character refusal added in c452ee55 does
not cover that path: a control character in a `wontfix_reason` still reaches
`yamlScalar`.

Why it is a nitpick rather than a defect: that path mints nothing, so it fails
atomically with no orphan draft — which is the consequence the sibling record
(iss-2608301206032013) existed to prevent. The refusal happens either way; only
its site and its message differ.

Left open for the facilitator. Closing it would mean routing derived grounds
through the same validator as supplied grounds, which is a small consolidation
in the direction the one-canonical-primitive rule already points.
