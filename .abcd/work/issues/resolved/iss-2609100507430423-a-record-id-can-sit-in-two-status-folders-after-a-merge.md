---
schema_version: 1
id: "iss-2609100507430423"
slug: "a-record-id-can-sit-in-two-status-folders-after-a-merge"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal (capture store, folder-as-status)"
resolution: "Every read of the ledger now refuses an id claimed by more than one record file, naming both paths: capture list (filtered or not), the bare capture board and abcd <iss-N> all route through the one check, which reads filenames across all three status directories and canonicalises a zero-padded twin onto its id. The two-folder case says plainly that folder membership IS the status, so the id has no defined status at all; the one-folder case says an id must name one record. The convention that stops it arising is stated in AGENTS.md: a record is resolved on the branch that carries it, never re-added to the default branch after a branch was cut from it."
impact: fix
---

A record id can end up in two status folders at once after a merge, and nothing flags it.

Observed landing 27 worker branches in a managed repository. Two issue records were committed to the default branch, in `open/`, after the worker branches had already been cut. Those branches then resolved the same issues, moving `open/` to `resolved/`. Git's rename detection did not pair the two sides — the file arrived on one side as an add and left on the other as a delete-plus-add at a different path — so the integration branch ended up carrying the record in BOTH folders. Folder-membership-as-status then said the same id was open and resolved simultaneously. It was caught by listing `open/` by hand and noticing a slug that had already been closed.

The store's whole status model rests on folder membership, and the one thing that model cannot survive is a record in two folders. That state is trivially detectable — it is a duplicate id across the store's own directories — and no verb or gate looks for it.

Wanted: `abcd capture list`, or a row in `abcd lint`, that treats an id present in more than one status folder as an error rather than rendering it twice or picking one arbitrarily. And a stated convention alongside it: a record is resolved on the branch that carries it, never re-added to the default branch after a branch was cut from it.

## Grounds

- pursued: the reader is where the gap was — findIssue already refuses a transition and record-lint already refuses a committed tree, and the board in between rendered the duplicate twice; it would be shown wrong if an ordinary status move refused, which the clean-ledger control asserts against
