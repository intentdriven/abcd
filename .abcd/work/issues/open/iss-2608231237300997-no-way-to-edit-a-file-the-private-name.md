---
schema_version: 1
id: iss-2608231237300997
slug: no-way-to-edit-a-file-the-private-name
severity: minor
category: documentation
found_at: ACKNOWLEDGEMENTS.md
found_during: user-observation
source: user-observation
remedy: "Waits on the owed ruling: if keyed anchoring, the operator refines the private entry with an explicit (^|[^[:alnum:]]) boundary so it stops matching the file's legitimate text, with no code change; if diff-scoped, the pre-commit hook refuses a staged blob only when an entry matches more lines in it than in the HEAD blob of the same path, still reading blobs rather than diff text; if acknowledged, a commit carrying a recorded acknowledgement that names the entry key passes and the acknowledgement is logged. Prove a code answer with a hook test where an edit away from an existing match passes and an added occurrence is still refused."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): For a reviewed edit past the private name guard: keyed entry anchoring, diff-scoped scanning, or a recorded per-commit acknowledgement?"
---

# There is no way to change text in a file the private name-guard blocks

The private banlist's `entry-14` refuses every commit touching
`ACKNOWLEDGEMENTS.md` on this machine, whatever the change is. The guard is
doing its job — the file carries a name that must not be published — but it
blocks the file rather than the offending text, so a change that removes text,
or edits a paragraph nowhere near the protected name, is refused exactly as a
change that would leak it.

That is the gap: a correct edit to a guarded file currently has no path at all.
The edit sits in the working tree or it does not happen.

## What raised it

The maintainer asked for the `## Inspirations` lead sentence to be removed
("Ideas and methodologies that shaped the design — not code abcd depends on."),
which renders on `/references/` as the paragraph above the inspiration entries.
The edit was made, refused by the guard, and — on the maintainer's instruction
— reverted. The sentence stands. The guard was not weakened and no workaround
was attempted.

## What a fix has to provide

A way to land a reviewed change to a guarded file WITHOUT weakening the guard.
Candidates, none chosen:

- Anchor the entry so it matches only the protected occurrence rather than the
  file, which `.abcd/.work.local/NEXT.md` already records as the outstanding
  step for `entry-14` (add `# abcd-banlist: keyed` to the private names file).
- Scan the DIFF rather than the file, so a change that neither adds nor keeps
  the protected text passes.
- An explicit, recorded per-commit acknowledgement for a change a human has
  read.

The first is the smallest and is already written down; the second is the one
that generalises, because it makes the guard's question the right one — does
this change publish the name — rather than a proxy for it.

## Remedy grounds (2026-09-29)

- Why: the record's three candidates, each with its change; the ruling is unanswered and none is picked. The keyed store format the body proposes to adopt already ships (internal/core/banlist/banlist.go), so the first candidate is an operator act on the private list alone. No answer lets an added occurrence through.
- Rejected: parsing git diff --cached text for the diff-scoped answer; the committed hook reads blobs because diff text left four ways to stage a banned name (internal/core/ahoy/defaults/pre-commit), and a count comparison between blobs keeps that property.
