---
schema_version: 1
id: iss-2608231237300997
slug: no-way-to-edit-a-file-the-private-name-guard-blocks
severity: minor
category: documentation
found_at: ACKNOWLEDGEMENTS.md
found_during: user-observation
source: user-observation
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: keyed entry anchoring vs diff-scoped scanning vs a recorded per-commit acknowledgement for a reviewed edit past the private name guard). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
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
