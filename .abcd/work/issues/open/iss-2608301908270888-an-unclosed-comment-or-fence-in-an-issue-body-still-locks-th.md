---
schema_version: 1
id: "iss-2608301908270888"
slug: "an-unclosed-comment-or-fence-in-an-issue-body-still-locks-th"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-fix-delta-ruthless"
found_at: "internal/core/grounds/record.go"
---

an unclosed comment or fence in an issue body still locks the record out of every triage verb and no open record tracks it

**This record is the marker the residual did not have.** It exists because
`iss-2608301803423101` -- a critical, self-declared branch-introduced defect --
was moved to `resolved/` with this consequence still live, and nothing in
`open/` mentioned it. The repository's own rule is that a fixed-but-open issue
leaves no marker to find it by; a not-fixed-but-closed one is the same fault
mirrored, and it is worse, because the record reads as settled.

What survives, at HEAD, reachable with ordinary prose and no malice:

```
abcd capture "the loader drops rules when the <!-- abcd-review marker is stale"
```

That text is accepted without a warning and written verbatim into the body.
Then all three triage verbs refuse, permanently:

```
resolve => grounds refused: the record's body leaves an HTML comment
           (`<!--` with no `-->`) open: the opener is body line 2, ...
wontfix => ... same
promote => ... same
```

There is no `edit`, `amend` or `reopen` verb. The only exit is a hand edit of a
committed file.

The delta made it SURVIVABLE and that was the agreed scope: the refusal now
names the construct and its line instead of blaming the grounds operand, and
`Promote` no longer mints a draft it will orphan. Body scope closed the
frontmatter half only. None of that closes this half, and the fix's own DECISIONS
entry concedes as much.

Two asymmetries worth recording for whoever takes it. `AppendToRecord` already
refuses an unclosed `<!--` in GROUNDS text, so the write side is inconsistent
with itself: it guards the operand and not the body it appends into. And
`Capture` validates nothing about a body it will later require to be parseable
-- the check belongs where the text enters, not where it is next appended to.

**RULED 2026-08-31: severity lowered to minor; the hand edit is accepted as the
repair path.** Three reasons, and the first is the one that matters.

The guard is CORRECT. `ParseSection` masks the body so a heading inside a fence
or comment is not mistaken for the real section, and the append verifies by
re-parsing. When an unclosed opener shadows the appended bullet, writing it
anyway would produce a record whose grounds no reader can see. Refusing is the
right answer; the defect was never the refusal.

The exit exists. These are committed markdown files in a git repository, the
refusal names the construct and the body line, and closing the comment is an
ordinary edit. The original framing — "the record can never leave open/" — is
true of the TOOL and not of the operator, and that overstatement was the
orchestrator's.

The exposure is latent and measured: 31 committed records carry a fence, 4 carry
a comment opener, and none is currently locked.

Rejected: validating bodies at capture time, which would close it at entry but
contradicts an adopted principle — "capture in particular must stay
frictionless" — and the moment recording a finding costs more than fixing one,
findings stop being recorded. Also rejected for now: a repair verb, which is the
honest answer if abcd's operator genuinely never opens git, but that is a
general "edit a record body" capability the design has withheld and it wants its
own intent rather than a fix bolted onto a shipped one.

Stays OPEN at minor: the tool still stops mid-workflow and hands the operator a
text editor, which is a real cost even if it is not a lockout.
