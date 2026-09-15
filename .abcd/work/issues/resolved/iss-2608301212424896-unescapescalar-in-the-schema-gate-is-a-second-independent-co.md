---
schema_version: 1
id: "iss-2608301212424896"
slug: "unescapescalar-in-the-schema-gate-is-a-second-independent-co"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "itd-179-round-2-ruthless"
found_at: "internal/core/lint/schema.go"
resolution: "the two byte-identical decoders are now one exported frontmatter.Unquote that capture's reader and record-lint's schema gate both call, and the grounds parity table carries a backslash-escaped row in both halves"
impact: internal
resolved_by:
  intent: "itd-179"
---

unescapeScalar in the schema gate is a second independent copy of capture.unquote and the parity table has no escaping row

Found by the round-2 adversarial ruthless review of build/itd-179.

`internal/core/lint/schema.go:683` `unescapeScalar` is a second, independent
copy of `capture.unquote` (`internal/core/capture/parse.go:270`). The parity
table meant to bind the gate to the reader carries rows for `single quoted`,
`empty list`, `block spelled`, `empty string`, `bare null` and `well formed` —
but no escaping row, so the two decoders are not actually pinned to each other
on the dimension where they duplicate logic.

Repo law (one-canonical-primitive): "Found the same primitive in two places?
Flag for consolidation — never add a third copy."

The risk is precisely the class round 1 closed in 050f3366: if `capture.unquote`
gains a case the gate does not, they diverge silently and a record the reader
skips goes lint-green.

Remedy: export the decoder from one shared home, or at minimum add a
backslash-escaped row to the parity table so a divergence fails a row rather
than passing unnoticed.

## Grounds

- pursued: we expect two copies of one decoder to drift on the case neither table covers, and the parity table's missing escaping row over a duplicated escaping loop is exactly that gap
