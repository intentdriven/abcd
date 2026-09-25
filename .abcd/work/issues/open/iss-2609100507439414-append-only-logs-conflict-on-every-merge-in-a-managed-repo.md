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
found_at: ".abcd/work/DECISIONS.md, CHANGELOG.md (in a managed repo)"
related_intents: [itd-2609151138388536]
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Decision log as a folder of records: convert, offer, or new-only for managed repos?"
---

A managed repository's shared append-only files conflict on nearly every merge, and abcd propagates neither of the two remedies it has already adopted for itself.

Observed landing 27 worker branches through one integration branch in a single day. `.abcd/work/DECISIONS.md` is one file every branch appends to, so it conflicted on the first two merges and would have conflicted on every later one; the session fixed it by adding a `merge=union` attribute for that path to `.gitattributes` by hand, which is correct because the lines are independent and dated. `CHANGELOG.md`'s `[Unreleased]` section has the same shape and conflicted on four of the eight merges that carried an entry, but union is NOT safe there: it duplicates the `###` headings. That one needed a hand-written section-merging script, kept only in the session's local tier. The reporting session refined this afterwards, and the distinction is the point of the record: once the union attribute was in place the decisions log stopped conflicting entirely, while the changelog went on conflicting on most of the merges that remained, because union is the wrong remedy for it rather than an unapplied one. Two conflict classes that look identical at a glance therefore have different fixes, and a remedy that scaffolds only the attribute would close one and leave the other exactly where it was.

abcd has already answered both questions for its own repository and neither answer travels. iss-118 resolved by adding `merge=union` to abcd's own `.gitattributes` for `DECISIONS.md` and `ACKNOWLEDGEMENTS.md`; nothing in `ahoy install` or `prepare` writes that attribute into a repository abcd adopts, so every managed repo rediscovers the conflict and fixes it by hand or not at all. iss-2608220150157510 carries the per-change changelog fragment proposal for abcd itself; a managed repo needs the same thing and has even less standing to invent it locally.

Wanted, in either order: scaffold the union attribute for the decisions log at adoption time (it is a one-line write into a file `ahoy` already manages), and give the changelog a per-change fragment directory, which removes both conflict classes rather than one. Storing decisions one-per-file like issues would do the same for the first, at the cost of a new id family — the ledger's one-file-per-record shape was the one thing in this run that never conflicted at all, in 27 merges and 33 resolutions.

## Grounds

- pursued: the two files that conflict on every merge are the two that are single append-to-the-bottom files, and the five record families that never conflicted in 27 branch merges are all one-file-per-entry — so the conflict is a property of the file shape, not of the content, and giving the decisions log the ledger's shape removes the class rather than patching it. What would show this wrong: a folder of minted decision files that still conflicts on merge (an index assembled into one committed file would do it, if the assembly is committed rather than derived), or a measured cost of reading the decision history that the index does not recover.

**Corroboration (2026-09-19, Gropius session gropiusllm-66, relayed to
abcd-17).** The conflict became a livelock at scale: a merge queue plus some
twenty pull requests that each append to DECISIONS.md and CHANGELOG.md. The
forge ignores the union merge driver, every landing re-dirties every other
entry, CI on every push saturates the runners, and the queue drops entries
that wait past its timeout. The session landed them as one integration branch
in one queue entry. Worth a line wherever the landing process is documented
for a managed repository, beside this record's deferral.

**Corroboration (2026-09-20, Gropius session gropiusllm-2b, relayed to
abcd-17).** Two further facets from a sub-agent-lane experiment in the same
repository. Inside one checkout: two sessions appending to DECISIONS.md at once
have no lock, and keep the file consistent only by an append-only discipline
agreed by message; the session asks for an append verb (`abcd decide --line
"…"` in shape) or for the log to become a directory of dated files, so
concurrent sessions are safe by construction rather than by agreement. Across
branches: the livelock recorded on 2026-09-19 recurred, and the
integration-branch landing that resolves it is folklore in that repository's
handover file; the session asks for a `launch integrate` verb or a documented
recipe so the pattern is abcd's rather than each repository's.

Same day, second session (gropiusllm-97): the two-session append to
DECISIONS.md and NEXT.md held by convention only, and the ask is the same
append-only `decide line` verb.
