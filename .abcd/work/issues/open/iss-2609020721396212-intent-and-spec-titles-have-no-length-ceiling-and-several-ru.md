---
schema_version: 1
id: "iss-2609020721396212"
slug: "intent-and-spec-titles-have-no-length-ceiling-and-several-ru"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/create.go"
---

Intent and spec titles have no length ceiling, and several run to a full sentence that carries the description rather than a name (the longest today are well over a hundred characters), so a roster, a status board, a changelog line and the record export on the website all show a paragraph where a title belongs, and the id-plus-slug filename derived from the title is truncated at a fixed width that cuts mid-word. Two changes: a threshold on the title at mint time (intent new, spec new, capture promote), refused with the excess named and the description offered as the place for the rest, with a size stated in the record schema and enforced by record-lint for hand-written records; and the website's record export truncating an existing over-long title at the display layer with an ellipsis and the full text on hover or on the record page, so the records already in the corpus do not have to be rewritten to render. The ceiling itself is a decision (a fixed character count, or a word count); the site truncation is not.

Measured on 2026-09-02 across every intent and spec: the four longest first-line headings run to 1553, 1521, 1512 and 1354 characters (itd-154, itd-101, itd-104, itd-103); they are press-release paragraphs written where the title goes, which is the shape any ceiling has to refuse and the site has to truncate.
