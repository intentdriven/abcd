---
schema_version: 1
id: "iss-2609100507431858"
slug: "nothing-lists-which-open-issues-already-carry-a-decision"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (capture list)"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Decided/blocked-on-decision signal in capture list --open: a frontmatter field or a recognised heading?"
---

No status surface says whether an open issue already carries a decision, so triage means reading every record in full.

Observed planning an autonomous sweep over a managed repository's 48 open issues. Some records ended in a dated "Decision" section, some in an "Interview outcome", some in an explicit call for the maintainer, and most in nothing at all. `abcd capture list --open` renders id, state, severity and slug, none of which distinguishes an issue whose fix is mechanical and agreed from one that is still waiting on a design call. The only way to sort them was to open all 48 and read the bodies, which is precisely the work a listing exists to avoid, and which does not scale to the ledger size the tool is otherwise happy to grow.

The distinction matters most exactly when the reader is an agent picking work to do unattended: an undecided issue handed to a worker becomes a worker making the decision, which is the thing the interview convention exists to prevent.

Wanted: a `decided` or `blocked-on-decision` field, or a recognised heading the record already carries in prose, that `abcd capture list --open` can render as a column. A recognised heading is the cheaper of the two and needs no schema change, since the convention is already being followed by hand in the bodies; a field is stronger, because a heading nobody wrote is indistinguishable from a decision nobody took.
