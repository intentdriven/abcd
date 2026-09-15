---
schema_version: 1
id: "iss-2608301918362294"
slug: "a-prose-enumeration-of-where-a-value-is-copied-cannot-be-mai"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "itd-179-claim-fix-review"
found_at: ".abcd/development"
---

a prose enumeration of where a value is copied cannot be maintained and has now been wrong four times on two different symbols

Found by the itd-179 claim-fix review, on a correction the ORCHESTRATOR wrote,
which is the point: the correction introduced a fresh false claim in the act of
fixing one, for the third consecutive iteration on this symbol.

The lineage on `grounds.Vocabulary` alone:

1. the comment claimed it was "the ONE copy every gate and every flag
   description reads"; no surface read it at all;
2. the correction enumerated five code sites (iss-2608301836222858);
3. the orchestrator's correction said six and asserted "the enumeration above
   MISSES a copy", singular;
4. it is at least SEVEN code sites plus two unlisted doc sites --
   `internal/core/intent/ready.go`'s `groundsRemedy` (live, printed to the
   operator by the readiness gate), `commands/capture.md` and
   `internal/README.md`. A tree sweep finds twelve files carrying a literal
   spelling.

And `groundsRemedy`'s own doc says it "is the one spelling of how a ground is
recorded, so the gate and the surfaces name the same command and the same closed
vocabulary" -- which the four `cli.go` spellings falsify. A "one copy" claim
with copies, inside the file that holds one of them.

**The same shape reached the same diagnosis on a different branch the same day.**
iss-2608301901264848 is a nine-store enumeration of reader behaviour that took
four turns and was wrong at the fourth. Two independent claim families, two
branches, four turns each, one remedy:

**A prose enumeration of facts about the codebase cannot fail, so it cannot be
maintained.** It goes stale silently and is believed precisely because it is
specific, and each correction is itself an unverifiable claim, which is why the
error rate does not fall with effort.

Remedy, deliberately NOT applied tonight: make the enumeration executable. A
test that sweeps the tree for literal spellings and asserts the set, so a new
copy fails a gate instead of ageing a record. That is the same remedy dispatched
for the sibling on the other branch.

It is not applied here because it needs a judgement this record cannot make: the
package that DEFINES the tokens legitimately spells them, so "what counts as a
copy" is a scope decision, not a grep. Making that call unsupervised, on the
second stop-condition hit of one session, is exactly what the stop exists to
prevent.

