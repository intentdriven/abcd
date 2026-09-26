---
id: spc-2609230613206890
slug: glossary-pointers-lint-and-mental-model
intent: itd-2609211913453478
origin: researcher-authored
production_mode: hand-written
---
# Glossary pointers, the family lint and the mental model

## Summary

The remainder of [itd-2609211913453478](../../intents/shipped/itd-2609211913453478-one-page-in-the-glossary-maps-abcd-s-record-families-and-how.md)
that [spc-2609212131112235](spc-2609212131112235-one-page-in-the-glossary-maps-abcd-s-record-families-and-how.md) did not deliver. spc-2609212131112235 closed on
2026-09-23 with acceptance criterion 3 delivered and criteria 1, 2 and 5 in part: the seven-family table in `glossary/core/record-families.md`, the `bundle` and `step` entries, `batch` defined on the page, the superseded `phase`, `milestone` and `roadmap` entries, and adr-9 marked superseded. This spec carries what did not ship.

## Scope

- **Acceptance criterion 1 (the missing half).** Every other glossary entry points at the record-families page. At closure only 5 of 36 glossary files reference it.
- **Acceptance criterion 2 (the missing half).** The phase documents carry a retirement line. At closure only `roadmap/phases/README.md` does; the `phase-0` to `phase-8` documents do not.
- **Acceptance criterion 4 (missing).** Record lint refuses an entry whose `not_to_be_confused_with` names nothing on the page, and reports a record frontmatter key naming a family the page does not define. No such rule exists at closure.
- **Acceptance criterion 5 (the missing half).** The brief's mental-model chapter reads brief, intent, spec (then steps) with the bundle and the derived release named. At closure the opening paragraph of the brief's [Mental Model](../../brief/01-product/03-mental-model.md#mental-model) chapter still describes four layers with phase.
