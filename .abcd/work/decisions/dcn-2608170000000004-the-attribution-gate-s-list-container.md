---
id: dcn-2608170000000004
slug: the-attribution-gate-s-list-container
date: 2026-08-17
---
2026-08-17 — The attribution gate's list-container fence limit is ACCEPTED as a
standing residual (iss-270, wontfix), not deferred. A fence opened inside a
first-level list item sits at column 2 (ordered: column 3), under the matcher's
absolute three-space limit, so it is read as document-level and a footer inside
that span is stripped although the forge renders it as an ordinary paragraph.
All three remedies are worse than the hole: tracking container context means a
markdown block parser inside a bash gate; refusing to strip from any document
containing a list marker is safe by construction but would disable the fence
concession in almost every real body here, reverting iss-268 in practice while
appearing to keep it; and forcing openers to column 0 was tested by the security
review and LEAKS, because under-recognising an opener is not conservative —
markdown then closes its block at a line the matcher treats as the opener,
shifting the whole span. Acceptance is defensible because the gate's defended
property is untouched: it exists for a footer a tool appends BY DEFAULT (itd-91,
78 pull requests), a tool appends last, and footer-last is caught whatever
precedes it — fuzzed at 0 violations in 2400 differential bodies against a real
CommonMark parser. This shape needs a body CONSTRUCTED AROUND the footer, which
the design already declines to defend, and such an author would delete the
footer instead. Pinned in the corpus for bulleted and ordered items so a change
is visible. Rejected: all three remedies above; and leaving the record open,
which would invite the indent-relaxing "fix" that widens the hole. Reopen only
if the gate is asked to defend against a deliberate author — a different design.
