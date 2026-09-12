---
schema_version: 1
id: "iss-2609100507439414"
slug: "append-only-logs-conflict-on-every-merge-in-a-managed-repo"
severity: "major"
category: "process"
source: "user-observation"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-09/10"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "abcd answered both halves of this for its own repository, with a merge attribute for the append-only decision log and a derived changelog rather than a hand-edited one, and neither answer travels to a repository it manages. Whether adoption should write those answers into a managed repo, and which of them, is a decision about how opinionated adoption is allowed to be. Scaffolding a merge attribute into somebody's repository unasked is exactly the kind of act this project's own principle about the user's directory tells it to think twice about."
found_at: ".abcd/work/DECISIONS.md, CHANGELOG.md (in a managed repo)"
---

A managed repository's shared append-only files conflict on nearly every merge, and abcd propagates neither of the two remedies it has already adopted for itself.

Observed landing 27 worker branches through one integration branch in a single day. `.abcd/work/DECISIONS.md` is one file every branch appends to, so it conflicted on the first two merges and would have conflicted on every later one; the session fixed it by adding a `merge=union` attribute for that path to `.gitattributes` by hand, which is correct because the lines are independent and dated. `CHANGELOG.md`'s `[Unreleased]` section has the same shape and conflicted on four of the eight merges that carried an entry, but union is NOT safe there: it duplicates the `###` headings. That one needed a hand-written section-merging script, kept only in the session's local tier. The reporting session refined this afterwards, and the distinction is the point of the record: once the union attribute was in place the decisions log stopped conflicting entirely, while the changelog went on conflicting on most of the merges that remained, because union is the wrong remedy for it rather than an unapplied one. Two conflict classes that look identical at a glance therefore have different fixes, and a remedy that scaffolds only the attribute would close one and leave the other exactly where it was.

abcd has already answered both questions for its own repository and neither answer travels. iss-118 resolved by adding `merge=union` to abcd's own `.gitattributes` for `DECISIONS.md` and `ACKNOWLEDGEMENTS.md`; nothing in `ahoy install` or `prepare` writes that attribute into a repository abcd adopts, so every managed repo rediscovers the conflict and fixes it by hand or not at all. iss-2608220150157510 carries the per-change changelog fragment proposal for abcd itself; a managed repo needs the same thing and has even less standing to invent it locally.

Wanted, in either order: scaffold the union attribute for the decisions log at adoption time (it is a one-line write into a file `ahoy` already manages), and give the changelog a per-change fragment directory, which removes both conflict classes rather than one. Storing decisions one-per-file like issues would do the same for the first, at the cost of a new id family — the ledger's one-file-per-record shape was the one thing in this run that never conflicted at all, in 27 merges and 33 resolutions.
