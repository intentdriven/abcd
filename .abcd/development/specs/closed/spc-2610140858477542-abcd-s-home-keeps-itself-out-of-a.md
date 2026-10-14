---
id: spc-2610140858477542
slug: abcd-s-home-keeps-itself-out-of-a
intent: itd-2610140858048004
origin: researcher-authored
production_mode: hand-written
---
# abcd-s-home-keeps-itself-out-of-a

## Summary

spc-2610140858477542 delivers itd-2610140858048004: abcd's home ignores itself,
once, so a dotfiles repository in the person's home never stages what abcd keeps
there.

**The lay.** A new package, `internal/core/homeignore`, owns one function,
`Lay(home)`. It opens an existing `~/.abcd.noindex` through the home-scope walk
(`fsutil.OpenHomeScope`), which refuses a symlinked home folder, and never
creates it: a home abcd has not made yet is left alone, and the command that
makes it is followed by a lay. Inside it, when no marker stands, it creates
`.gitignore` holding `*` and then the marker `gitignore-laid`, both owner-only
and both by exclusive create, so it never overwrites a file. A `.gitignore` the
person already keeps there is left as it is, and the marker is written beside
it. The marker is plain text that says what the file is for and that deleting
`.gitignore` is how a person lets a repository in their home track this folder.
`*` matches both files, so neither is ever staged.

**Once, and never again.** With the marker standing, `Lay` does nothing, so a
`.gitignore` the person deleted stays deleted. A home from before this change has
neither file, and its next command lays both.

**Where it runs.** The CLI front door (`run` in `internal/surface/cli/cli.go`)
calls it twice: before the command, for a home that already exists, and after
it, for a home the command has just created. It is skipped while the old home's
stop is in force, for an unset or relative HOME, and for the status line, which
writes nothing at all by the product thinker's ruling of 2026-10-05
(iss-2610050556383525). A failure never fails the
command: it prints one line on stderr naming why the file was not laid, except
under `--json` and in a hook call, where a machine reads stderr, and the next
command tries again. A symlinked `~/.abcd.noindex` reads as absent and is never
laid. It reads nothing of, and runs nothing in, any
repository.

**What does not change.** The credential store keeps refusing the abcd home
inside any git working tree, the account's own included, whatever the file says
(adr-2610140859394097 records both rules). A private ideate verdict keeps the read rule
(`gitutil.HomeInsideWorkingTree`): allowed in the account's own git-tracked
home, which the laid file keeps out of that repository, and refused in a HOME
planted inside another checkout.

**Acceptance criteria.** Each Given/When/Then of the intent maps to a test:
`homeignore` unit tests for the lay, the never-again rule, an existing
`.gitignore`, a symlinked home and a failed write; a CLI test for the front
door's before-and-after lay and its stderr line; and a git test that runs
`git add -A` in a home that is a working tree and finds nothing under
`~/.abcd.noindex` staged.

## Footprint

- packages: internal/core/homeignore, internal/surface/cli
- tests: internal/core/homeignore/homeignore_test.go, internal/surface/cli/homeignore_test.go, internal/core/credential/homeignore_test.go, internal/core/ideate/visibility_test.go

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
