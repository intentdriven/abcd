---
schema_version: 1
id: "iss-2608301423540137"
slug: "the-orchestrator-committed-record-bodies-to-three-build-bran"
severity: "minor"
category: "lapse"
source: "user-observation"
found_during: "itd-189-round-3-builder"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/issues"
lapsed_at: "2026-08-30T12:37:00Z"
---

the orchestrator committed record bodies to three build branches without running the record and site gates, turning site-render red on two of them

The lapse, stated plainly: the orchestrator ran `record-lint`, `docs-lint` and
`site-render` before every commit on the integration branch and on the
followups branch, and ran NONE of them before the capture commits on
build/itd-183, build/itd-179 and build/itd-189. It relied on the pre-commit
hook, which runs the name-guard alone.

What it cost. In `070e1bd0` a two-line sample of gate output was indented four
spaces inside iss-2608301308367566. Markdown reads that as an indented code
block, and this repo's site renderer supports a fixed markdown subset that has
none, so `site-render` exited non-zero and took `make preflight` with it — in
EVERY checkout of build/itd-189 from that commit onward. The itd-189 round-3
builder hit it, diagnosed it, and correctly judged it pre-existing to its own round;
its brief had told it to expect exit 0 under the HOME alias, so the brief was
stale and the builder had to work that out for itself.

A sweep for the same shape found one more, iss-2608301237456350 on
build/itd-183, six indented lines. That branch's round-10 builder was mid-round,
so the fix was sent to it as a message rather than edited under a running agent.
build/itd-179 and the followups branch are clean.

Two things this is NOT. It is not a defect in any branch's work. And it is not
the same finding as iss-2608301350287219, which records the real gap — that
nothing between writing a record and pushing it reads the body AS markdown, so
the first reader is the site render at the far end of the gate. That gap is why
the mistake was possible; this record is that the orchestrator had the gate
available, uses it everywhere else, and skipped it here.

Remedy, adopted immediately: a capture commit on a build branch runs the same
three gates as a commit anywhere else, by exit code, before committing. The
tier of the branch does not change what the record has to survive.
