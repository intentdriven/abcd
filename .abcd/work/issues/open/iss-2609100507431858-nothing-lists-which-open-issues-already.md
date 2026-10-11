---
schema_version: 1
id: "iss-2609100507431858"
slug: "nothing-lists-which-open-issues-already"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (capture list)"
deferred_after: "v0.11.1"
deferral_reason: "planning F owed to the product thinker: capture list --open still renders no decided or blocked-on-decision column. A frontmatter field or a recognised heading is the choice owed. (re-checked at e792a2314 by lane drainDQ3, run A, 2026-09-29)"
remedy: "Waits on ruling F (a field or a recognised heading): if a field, use the remedy: field that ruling BX3 makes every new issue carry, and have capture list --open render a column reading decided when a remedy is set, waiting when it opens with Waits on, and none when absent; if a heading, recognise a dated Decision heading in the body under the same column instead; either proven by a list-rendering test over three fixture records."
---

No status surface says whether an open issue already carries a decision, so triage means reading every record in full.

Observed planning an autonomous sweep over a managed repository's 48 open issues. Some records ended in a dated "Decision" section, some in an "Interview outcome", some in an explicit call for the maintainer, and most in nothing at all. `abcd capture list --open` renders id, state, severity and slug, none of which distinguishes an issue whose fix is mechanical and agreed from one that is still waiting on a design call. The only way to sort them was to open all 48 and read the bodies, which is precisely the work a listing exists to avoid, and which does not scale to the ledger size the tool is otherwise happy to grow.

The distinction matters most exactly when the reader is an agent picking work to do unattended: an undecided issue handed to a worker becomes a worker making the decision, which is the thing the interview convention exists to prevent.

Wanted: a `decided` or `blocked-on-decision` field, or a recognised heading the record already carries in prose, that `abcd capture list --open` can render as a column. A recognised heading is the cheaper of the two and needs no schema change, since the convention is already being followed by hand in the bodies; a field is stronger, because a heading nobody wrote is indistinguishable from a decision nobody took.

## Remedy grounds (2026-09-29)

Why: rulings BX3 and H12 (2026-09-29) already give every issue a remedy line and a Waits on opening for a record that hands itself back, so the field option needs no new schema key. Rejected: a new decided or blocked-on-decision frontmatter key, which would duplicate what the remedy field already states.
