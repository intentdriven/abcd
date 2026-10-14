---
id: itd-2610141956319695
slug: a-deferred-finding-is-its-own-captured
spec_id: null
kind: discipline
kind_notes: "A rule every review inherits: a finding is fixed, shown invalid, or deferred to an open issue record of its own, and never folded into another finding's record. The release receipt gate (record-lint --release-gate, and abcd launch receipts through the same reader) enforces it from release-gate manifest schemaVersion 2."
suggested_kind: discipline
reclassification_history: []
builds_on: []
severity: major
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A deferred finding is its own captured record

## The rule

**A deferral is always a capture. There is no exception.** Every finding a
review confirms leaves the review in exactly one of three states:

- **fixed**: the change that answers it is in the reviewed tree;
- **invalid**: re-verification shows it is not a defect;
- **deferred**: it is captured as an open issue record of its own, and the
  finding names that record.

A deferred finding's record holds that finding and no other. Two findings that
are the same defect are one finding, recorded once; two different defects are
two records. A deferral is never written as prose: "accepted", "tracked by",
"knowingly shipped" and "carried to" are all the third word in disguise, and a
deferral that names no record of its own is not a deferral but a finding
dropped.

In a release receipt the three states are the three words of the finding's
`disposition`, spelled exactly, and a deferred finding carries a separate field
naming its record:

```json
{"where": "04-surfaces/16-lint.md:12", "disposition": "deferred", "record": "iss-<id>"}
```

## Why

The product thinker ruled it on 2026-10-14, in these words: "deferrals must
always be captures. no exception." and "capture this as an enforced abcd
discipline. No exceptions." The ruling followed a reading of the committed
release receipts under `.abcd/work/reviews/`.

Those receipts record 2,178 findings. 1,090 of them, across the receipts of 12
reviewed commits, name one record, iss-2609091956001547, the issue that says the
brief-surface crosscheck does not converge; the receipts of one commit alone
carry 150. That record is a true statement about the crosscheck, and as a home
for a finding it is a catch-all: once a thousand findings share it, resolving it
says nothing about any one of them, and none of them can be fixed, shown
invalid or re-asked on its own.

The dispositions themselves took about a dozen free-text shapes:
`deferred`, `deferred-pre-existing`, `deferred-release-introduced`,
`captured as …`, `tracked by …`, `accepted: …`, `accepted in part …`,
`fixed in …`, `invalid: …`, an instruction quoted as a removal, "out of scope
and non-blocking", and two left blank. Among the accepted ones,
"accepted: non-blocking, knowingly shipped unfixed" is a deferral by another
name, with no record behind it at all. The gate before this one asked only that
a disposition was not blank, so every one of those shapes passed.

## The gate

- **Given** a release receipt for the reviewed commit, in a tree whose
  release-gate manifest is schemaVersion 2 or later, **when** the release gate
  reads a finding, **then** its `disposition` is byte-for-byte `fixed`,
  `invalid` or `deferred`, and any other value, a capitalised word or a
  trailing space included, refuses the release.
- **Given** a `deferred` finding, **when** the gate reads it, **then** it names
  `record: iss-<digits>`, with no leading zero; the id names exactly one file in
  the issue ledger; that file sits in `open/`, is read with no symbolic link in
  its path, and is a record the ledger's own reader reads. A missing record, a
  resolved or wontfix one, an id two files claim, a linked file, and a record
  the ledger's reader refuses each refuse the release.
- **Given** a `fixed` or `invalid` finding, **when** the gate reads it, **then**
  it names no record.
- **Given** the receipts of every required gate for the release, **when** two or
  more deferred findings name the same record, **then** the release is refused
  once for that record, naming it, the count and each finding.
- **Given** a release-review capture, **when** the release is cut, **then** the
  capture is committed on the release branch after the receipts commit, so the
  merge tree the release job checks out holds it; `abcd launch receipts` refuses
  while it is uncommitted.

**What the gate does not check.** The gate checks that the record is open,
well-formed and named by one finding only. It cannot check that the record
describes the finding that names it, or that the record is new rather than an
older open issue about something else: both are judgements about meaning, not
shape. The reviewer who dispositions the finding owns them, and a record whose
text does not match its finding is an ordinary review defect, caught by reading.
Naming an older open record is correct when that record really is this finding:
a defect already captured is deferred to the record that holds it, not captured
twice.

## Fit

It **refines**
[pre-existing-is-not-a-defence](../../principles/pre-existing-is-not-a-defence.md).
That principle says a confirmed finding is fixed or deferred out loud, and that
a deferral is a decision recorded where the release can see it. This record says
what the recording is: a capture of its own, named by the finding. The
principle's two claims stand as written; this one closes the gap between
"recorded" and "recorded in a catch-all", which the principle's own words,
"filing is a decision to make no decision", already condemn.

It **composes with** the `deferred_after` grant. A deferred finding's record is
an ordinary open issue record, so `changelog.GuardFindings` judges it like any
other: a `major` or `critical` record that entered the ledger since the anchor
tag refuses the next cut unless it is fixed, wontfixed, or granted a one-cycle
`deferred_after` with its `deferral_reason`. This discipline decides where the
deferral is written; the grant decides how long it may stand.

## Staging

**Enforced at the release receipts from release-gate manifest schemaVersion 2.**
The committed manifest moved to schemaVersion 2 for the release after v0.14.1,
so that release is the first the rule judges. A commit re-gated from before it
reads its own schemaVersion-1 manifest and keeps the free-text dispositions its
receipts were written to, so no historical receipt is rewritten. A release after
v0.14.1 whose manifest is below schemaVersion 2 is refused, so the rule cannot
be turned off by setting the manifest back. The rule is armed only by the
release's arming of the receipt gate; the doc-fidelity gate,
which reads its receipts through the same reader with dispositions of its own,
is untouched.

The rule binds every review, and the release receipt is the only place it is
checked today. Not yet enforced, each a later rung:

- **Pull-request review threads.** A thread resolved as "deferred" or "follow
  up" names no record, and nothing reads it.
- **Owed items in a run log.** An autonomous run's log lists what it owes; an
  owed item is a deferral, and nothing holds it to a capture.
- **Peer sessions.** A finding one session reports to another is deferred in
  conversation, and becomes a record only when someone captures it.

Until those rungs exist, the rule there is the documented protocol: the finding
is captured, and the capture is named where the deferral is written.
