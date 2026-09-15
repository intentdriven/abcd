---
id: itd-4
provenance:
  - origin: ABCD-EVAL-REFUSED-NESTED-MAPPING
---

# Selection criteria

The excluded key above is nested inside a block-sequence entry. The field
reader reports one value per top-level key and never sees it, so there was no
span to redact; the floor now refuses the NESTING rather than learning one more
spelling of the key.
