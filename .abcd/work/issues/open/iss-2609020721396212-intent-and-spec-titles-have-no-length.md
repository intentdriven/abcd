---
schema_version: 1
id: "iss-2609020721396212"
slug: "intent-and-spec-titles-have-no-length"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/create.go"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker: is the title ceiling at mint a character count or a word count, and what value? An implementation at 120 characters (recordid.MaxTitleChars, record_title_length at warn over a listed corpus, site truncation) sits on the unmerged branch docs/housekeeping-titles-register-brief (b5c7d550c to 4c54ef5fa) and is relanded as its own lane once ruled."
remedy: "Waits on the product thinker's ruling on the title ceiling at mint, which the deferral names as owed (a character count or a word count, and what value): once ruled, refuse a longer title at intent new, spec new and capture promote, naming the excess and offering the press release for the rest; derive a promoted draft's H1 with the word-boundary cut quoted-text drafts already get (internal/core/intent/create.go) instead of the issue's whole first body line (internal/core/capture/promote.go); add a record-lint rule record_title_length at warn so the records already over the ceiling are named without blocking; proven by tests at the ruled bound on each mint path and a lint fixture just past it. The site's list truncation needs no ruling and may land first: cut an over-long title on a word boundary with an ellipsis, the record page keeping the whole title, proven by a render test of a long title. The earlier implementation the deferral cites (commits b5c7d550c to 4c54ef5fa) is not on origin, so the lane starts from the current tree. Grounds: internal, the record's own split between the ruled ceiling and the unruled truncation."
---

Intent and spec titles have no length ceiling, and several run to a full sentence that carries the description rather than a name (the longest today are well over a hundred characters), so a roster, a status board, a changelog line and the record export on the website all show a paragraph where a title belongs, and the id-plus-slug filename derived from the title is truncated at a fixed width that cuts mid-word. Two changes: a threshold on the title at mint time (intent new, spec new, capture promote), refused with the excess named and the description offered as the place for the rest, with a size stated in the record schema and enforced by record-lint for hand-written records; and the website's record export truncating an existing over-long title at the display layer with an ellipsis and the full text on hover or on the record page, so the records already in the corpus do not have to be rewritten to render. The ceiling itself is a decision (a fixed character count, or a word count); the site truncation is not.

Measured on 2026-09-02 across every intent and spec: the four longest first-line headings run to 1553, 1521, 1512 and 1354 characters (itd-154, itd-101, itd-104, itd-103); they are press-release paragraphs written where the title goes, which is the shape any ceiling has to refuse and the site has to truncate.
