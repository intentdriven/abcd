# One writer per file

**The rule.** A committed record survives parallel programmes only when it has
one writer per merge window, reached by exactly one of two shapes: a
multi-writer source is atomicised into per-record files — and the family is
complete only when its id mint is guarded by an armed uniqueness detector —
or a single file is kept only where a single writer is guaranteed by
construction (a derived aggregate with one publishing path, or a per-worktree
local file). A multi-writer single file is a merge hotspot regardless of how
disciplined its appenders are.

**Why.** The 2026-07-22 two-programme merge drew the line empirically. Across
roughly thirty record files landing from two concurrent programmes, the only
textual conflict anywhere was `.abcd/work/DECISIONS.md` — the one shared
record that is a multi-writer append-only file. Every atomicised family
(issues, specs, intents) merged file-clean, but collided on minted ids
exactly where no detector was armed: the issue collisions (iss-110, iss-111)
were caught mechanically by `issue_id_unique`, while the spec collisions
(spc-10, spc-11) were caught only by a human reading a merge diff — the gap
iss-115 records. Atomicisation without a detector trades a visible textual
conflict for a silent identity collision, which is strictly worse. The
changelog programme is the pattern's clearest positive instance: release
lines are atomicised into impact-carrying issue records (detector-guarded),
and the `CHANGELOG.md` that remains has exactly one writer — the ship path
on main that ADR-37 names.

**Bounds.**

- The two shapes are exhaustive for committed records. Per-worktree local
  files (`.abcd/.work.local/`) satisfy single-writer by construction and are
  out of scope.
- "Armed" means a blocker at a gate that runs on the merged tree (push gate
  or CI), not an advisory warning: the skew class only exists in the union
  of branches, so a branch-local check cannot see it.
- The rule names the shape, not the remedy. For an existing hotspot the
  smallest compliant fix may be a `merge=union` attribute (legitimate for an
  append-only ledger whose entries never need identity) rather than full
  atomicisation; the choice is a design call per record. The attribute holds
  for a local `git merge` only: the forge computes a pull request's
  mergeability and the merge queue's merge without it, so two open pull
  requests appending to the same file still conflict there. Counting that
  cost is what makes one file per entry the remedy that removes the conflict
  everywhere (adr-2609151138420062).

**Promotion.** The detector half is already discipline-shaped for issues and
intents (`issue_id_unique`, `intent_lifecycle` via the shared
`validateIDUnique` primitive); extending it to spec and ADR ids (iss-115)
and adding a lint that flags multi-writer single files under `.abcd/work/`
would promote the whole principle.
