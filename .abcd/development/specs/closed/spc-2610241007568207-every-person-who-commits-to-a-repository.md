---
id: spc-2610241007568207
slug: every-person-who-commits-to-a-repository
intent: itd-2610240918547728
origin: researcher-authored
production_mode: dictated-and-formatted
---
# every-person-who-commits-to-a-repository

## Summary

spc-2610241007568207 moves each person's commit identity out of the
repository and onto their own machine, so a second contributor is never told
to commit as the maintainer (iss-2610240830397793), while a commit under a
machine's identity, or under a mistyped email of one's own, is still refused.
It reverses the committed pin of itd-131; the decision is
[adr-2610241022541063](../../decisions/adrs/2610241022541063-each-person-s-commit-identity-lives-on.md).

### Scope

This lane is the identity half of the setup redesign: the per-machine record,
the commit check that reads it (Go and the committed shell gate), the
migration of a committed pin, the repository's production mode kept
committed, and the minimum wiring into `abcd ahoy install` so the gap and its
fix work today. The setup-flow lane builds on the core API below; it is not
in this lane.

### Approach

- **The record** (`internal/core/identity/record.go`). `{"name","email"}` under
  the user-level home, read most specific first:
  `~/.abcd.noindex/identity/<root-sha>.json` (one repository, keyed on its
  root commit) then `~/.abcd.noindex/identity.json` (the machine-wide default).
  Read through `fsutil.ReadHomeDeclaration` (owned by this account, not
  writable by another, 4096-byte cap, refused behind a symlinked
  `~/.abcd.noindex`); a record that is there and fails a guard or does not
  parse is `ErrRecordRefused`, never "no record". Written atomically through
  `fsutil.EnsureHomeScope` at `abcdhome.DirMode`, the file at
  `abcdhome.FileMode`; a machine identity, an incomplete one or one the shell's
  sed could not read back is refused.
- **The check** (`identity.Check`, `abcd ahoy --identity`). Holds the author
  and committer to the record that applies; refuses a machine identity
  (`tool-identities.txt`) in either role whether or not a record exists; with
  no record a human identity passes (`StatusNoRecord`). Nothing the repository
  commits is read.
- **The shell gate** (`.githooks/pre-commit`). Self-contained: reads the same
  two files from `$HOME` in the same order, with shell equivalents of every
  guard (symlinked folder or file, not regular, not owned, group- or
  other-writable file or folder, over 4096 bytes, unparseable), each a BLOCK;
  refuses a machine identity from the one list, found beside the hook's own
  checkout or in the working tree (a copied hook with no list warns in one
  line and checks the record alone); names the person's own record and how to
  change it in every refusal.
- **The migration** (`internal/core/identity/migrate.go`). `PlanMigration`
  reads `.abcd/config/identity.json`; the owner is the person whose git
  identity (repo-resolved or global) is the stored one. `ApplyMigration` writes
  the owner's record for the repository (never over a different one), moves
  `production_mode` to `.abcd/config/provenance.json`, and removes the pin; it
  refuses a non-owner and a pin holding a member it cannot carry.
- **The production mode** (`internal/core/provenance/declared.go`).
  `provenance.DeclaredMode` reads `.abcd/config/provenance.json`, else the
  legacy pin's `production_mode`, else `hand-written`; the CLI's
  `resolveProductionMode` calls it.
- **Ahoy** (`internal/core/ahoy/detect.go`, `identity_establish.go`).
  `git_identity.unrecorded` and `git_identity.repo_pin` are `machine-identity`
  gaps, never behind a category approval, asked only at a terminal as their
  own questions (`stepIdentityMigrate`, `stepIdentityRecord`), listed in
  `optional_skipped` otherwise. A contributor's `repo_pin` gap is not
  resolvable and names no one. The itd-131 propose-and-confirm step proposes
  the person's own record, else their global identity.

### How each criterion is met

1. **No record, at a terminal.** `git_identity.unrecorded` is raised;
   `stepIdentityRecord` proposes `identity.ProposeRecord` (global identity, else
   the repo-resolved one, never a machine's) as its own question and writes
   the machine-wide record on a yes. Tests:
   `TestStepIdentityRecord_RecordsOnYesAtATerminal`,
   `TestStepIdentityRecord_WritesNothingOffATerminalUnderYesOrOnNo`,
   `TestInstallRecordsTheIdentityAtATerminal`,
   `TestAhoyInstallPipedAnswersNeverRecordTheIdentity`.
2. **A contributor who is not the maintainer.** Neither the check nor the
   shell gate reads the committed pin; the contributor's `repo_pin` gap says
   only that the stored identity is not theirs. Tests:
   `TestCheck_IgnoresACommittedPin`,
   `TestDetectGitIdentity_ContributorIsNeverShownTheMaintainer`, the hook case
   "a committed pin of the maintainer does not refuse a contributor".
3. **A different email, a bot's or an AI tool's name.** Refused by the shell
   gate and `ahoy --identity`, naming the person's own record and how to
   change it. Tests: the hook cases "mismatch blocks, naming the record…",
   "an AI tool's identity blocks…", "a bot's identity blocks…",
   `TestAhoyIdentityFlagHoldsTheCommitIdentityToThePin`,
   `TestDetectGitIdentity_Mismatch`.
4. **A repository that stores an identity today.** On the owner's machine, at
   a terminal, `stepIdentityMigrate` moves it to their record, keeps
   `production_mode` in `.abcd/config/provenance.json`, removes the pin and
   notes each; on another person's machine nothing is copied or removed.
   Tests: `TestMigration_OwnerMovesThePin`,
   `TestMigration_NonOwnerIsNotGivenThePin`,
   `TestMigration_UnknownMemberBlocks`, `TestMigration_KeepsADifferentRecord`,
   `TestStepIdentityMigrate_MovesThePinOnYes`,
   `TestDeclaredModeFallsBackToTheLegacyPin`.
5. **Unattended, fresh machine, no record.** A human identity the launcher
   supplied commits (`StatusNoRecord` does not block; the hook's "no record
   passes a human"); a machine identity is refused in either role
   (`TestCheck_FlagsAToolIdentity`, `TestBlocks`, the hook's machine cases).
   Off a terminal setup records nothing and says so.

### Known limits

- A repository with no git identity configured anywhere: git invents one
  from the account and host name and exports it into the hook, where it is
  indistinguishable from a supplied one; the Go check reports it as
  `git_identity.unset` (advisory until a record applies).
- The shell gate refuses a group-writable folder the Go reader admits when the
  group is the owner's own: stricter, in the fail-closed direction.

## Footprint

- packages: internal/core/identity, internal/core/provenance,
  internal/core/ahoy, internal/surface/cli
- hooks: .githooks/pre-commit (the self-contained identity gate)
- docs: commands/ahoy.md, commands/capture.md, commands/intent.md,
  .abcd/development/brief/04-surfaces/01-ahoy.md, .github/CONTRIBUTING.md,
  docs/reference/terminology.md, docs/reference/cli/commands.md (generated)
- records: adr-2610241022541063, iss-2610240830397793
- tests: internal/core/identity (record, migrate, hook, committer, env),
  internal/core/provenance (declared), internal/core/ahoy (identity gap,
  establish, apply, summary, swallowed writes), internal/surface/cli (ahoy
  identity gate, piped install, consolidate, capture production mode)

## Steps

_No steps listed: the spec is built as one step. To split it, list the steps in order as `1. <title>`, each with `- packages:` and `- tests:` indented beneath it; `- landed: <pull request or commit>` marks a step that has landed._
