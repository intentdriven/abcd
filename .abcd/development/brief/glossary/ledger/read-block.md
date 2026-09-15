---
term: read-block
bounded_context: ledger
definition: The wall that keeps ledger content from a cold reading: positive inclusion at the assembler, field projection out of files that hold both cold and warm material, a manifest that enumerates what was passed, and an eval that fails when warm material reaches a reading.
aliases: ["read block", "firewall"]
forbidden_synonyms: ["access control", "permission"]
status: draft
introduced_in: itd-183
starts_when: null
ends_when: null
not_to_be_confused_with: ledger/warm
versions: null
---

# read-block

The **read-block** is held by mechanism rather than asserted. Exclusion is positive: the assembler names what it includes, so a record type added later is excluded by default. The instrument's own outputs, manifests and reading records, sit inside the read-block on the next run. The scribe's context is its exact inverse: ledger content and never the shipped tree.

## When to use

Use it for the assembler's include table and exclusion floor, the manifest's exclusion assertions, and the eval that falsifies them.

## When NOT to use

Do not call the read-block an access control: the location tiering is organisational, and nothing but the assembler enforces the block.

## Related terms

- [warm](warm.md)
- [cold-reading](cold-reading.md)
