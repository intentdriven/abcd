---
schema_version: 1
id: "iss-2608301803423101"
slug: "an-unclosed-comment-or-fence-anywhere-in-an-issue-record-bli"
severity: "critical"
category: "bug"
source: "user-observation"
found_during: "itd-179-delta-ruthless"
found_at: "internal/core/grounds/record.go"
resolution: "the refusal now names the line and the construct that blinded the record instead of the grounds operand, and promote establishes that the record can accept the append before it mints, so a permanent refusal no longer leaks one orphan draft per attempt"
impact: fix
resolved_by:
  intent: "itd-179"
---

an unclosed comment or fence anywhere in an issue record blinds every grounds triage route permanently and promote orphans a draft per attempt

Found by the itd-179 delta review. BRANCH-INTRODUCED, verified: the same inputs
succeed at `d1731004`.

The readback guard masks the WHOLE record, frontmatter included, and
`mdrecord.Mask` runs an unclosed `<!--` or an unclosed fence to end of file --
correctly, per CommonMark and per its own doc. That is safe for a hand-authored
intent record. It is not safe for an ISSUE record, whose body is verbatim
operator prose that `Capture` never validates.

So an ordinary sentence bricks the record. Captured text containing an unclosed
comment marker is written happily, and then every grounds-bearing route refuses,
permanently:

```
resolved: grounds refused: the appended grounds entry does not read back
(0 entries after the append, expected 1); nothing written
```

Three consequences, in order of severity:

1. **The record can never leave `open/`.** Resolve, wontfix and promote all
   refuse, so there is no triage path at all. Only a hand edit of the committed
   file clears it.
2. **`Promote` mints before it appends, so each attempt ORPHANS A DRAFT.** The
   reviewer ran it three times and got itd-1, itd-2, itd-3, with the draft
   counter climbing each time, and the repair command the message names
   (`abcd capture promote iss-N --intent itd-1`) fails identically.
3. **The message blames the wrong text.** The operator's grounds entry is
   innocent; the fault is a construct elsewhere in the record. It sends them to
   rewrite the one thing that is fine. That is this cycle's standing class
   again, now as a misattribution rather than a false mechanism.

Reachable through frontmatter the tool itself writes, not only through the body:
`--found-at "internal/core/lint (the <!-- marker scan)"` at capture time, and
`--resolution "... strips the <!-- abcd-review marker ..."` at resolve time,
because `transition` sets the note BEFORE it appends.

Nothing in-tree is bricked today -- zero committed records carry `<!--` -- so
the class is forward-facing.

Remedy has two halves and both are needed. The refusal must DIAGNOSE rather than
misattribute: `Mask` already knows which line opened the span, so name that line
and that construct. And `Promote` must establish that the append can succeed
before it mints, or a permanent refusal leaks one orphan draft per attempt.

## Grounds

- pursued: we expect a locked record to be diagnosable from its refusal alone, and a refusal that can never succeed to leave no residue behind it
