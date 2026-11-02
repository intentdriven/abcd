---
id: itd-2610021503208208
slug: in-an-abcd-managed-repository-the-hooks
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# Personal git hooks keep running in abcd-managed repositories

## Press Release

> In an abcd-managed repository, the hooks a user keeps for all their own repositories (a commit-message linter, a secret scan before a push) keep running next to abcd's, because abcd's committed hooks run the repository's own check first and then hand over to the user's personal hook of the same name, from one folder the user names once in their global git config.

## Why This Matters

Alice keeps a few git hooks of her own for every repository she works in: a commit-message linter and a secret scan before each push. `abcd ahoy` points a managed repository's `core.hooksPath` at its committed `.githooks/`, and git runs exactly one hooks folder: a repository's own `core.hooksPath` replaces her global one, it does not add to it. So in every abcd-managed repository her personal hooks stop running, and nothing tells her.

Her workarounds are poor. Copying her hooks into each repository's `.githooks/` commits her personal tooling into other people's repositories. A global dispatcher folder (a global `core.hooksPath` whose hooks forward to each repository's hooks) is undone by the very per-repository setting abcd makes, and it is easily mistaken for real hooks: on 2026-10-02 an agent reported that adopting abcd's hooks had dropped five global hooks, when the global folder held only forwarding scripts.

abcd already separates what the repository owns from what one machine adds (the committed name guard and the private banlist). Personal hooks are the same split: the repository's checks are committed and run first; the user's own hooks live on their machine and run after.

## Mechanism

We expect a user's personal hooks to run in every abcd-managed repository because git calls exactly one file per event in the active hooks folder: if abcd's `.githooks/` holds a file for every client-side event a user may hook, and each file ends by running the user's same-named hook (with the same arguments and the same stdin), then wherever abcd's hook runs, the user's runs too, without any global `core.hooksPath`.

## Scope Conditions

- Repositories whose hooks abcd manages through `core.hooksPath .githooks`, set per clone by `abcd ahoy`.
- Client-side git hooks only; server-side hooks are out of scope.
- The user's hooks folder is on the same machine and is named only in the user's global git config; nothing about it is committed.
- Commits and pushes that git runs no hook for (a fast-forward pull, a rebase, `git am`, a skipped-hooks commit) stay unhooked, as today.

## Acceptance Criteria

- **Given** Alice has named a personal hooks folder once in her global git config and it holds an executable `commit-msg`, **when** she commits in an abcd-managed repository, **then** the repository's own `commit-msg` check (if any) runs first, her `commit-msg` runs after it with the same message-file argument, and a non-zero exit from either stops the commit.
- **Given** Carol's personal folder holds a `pre-push`, **when** she pushes several refs at once, **then** her hook receives exactly the ref lines on stdin that the repository's own pre-push check received, and her hook's failure stops the push.
- **Given** Bob has named no personal hooks folder, **when** he commits, merges or pushes in an abcd-managed repository, **then** every hook behaves exactly as before this change, with no extra output.
- **Given** Alice's personal folder has no hook for an event, **when** that event fires, **then** only the repository's own check runs and nothing is reported about the missing personal hook.
- **Given** Alice's setting names a folder that does not exist, or a same-named file that is not executable, **when** a hook fires, **then** the hook says so on stderr naming the setting, never silently.
- **Given** a personal hook exits 0, **when** the repository's own check failed first, **then** the commit or push is still refused: a personal hook can add a check, never lift one.
- **Given** Alice runs `abcd ahoy`, **when** her global git config sets `core.hooksPath` and the repository sets its own, **then** ahoy reports that the repository's setting overrides her global hooks and names the personal-hooks setting that keeps them running.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [forge, CI and merge SOTA note](../../research/notes/2026-10-23-forge-ci-merge-sota.md).

- **`core.hooksPath` replaces the hooks directory; it does not add to it**
  (git-config: hooks are found "in that directory ... instead of in
  `$GIT_DIR/hooks`"). The intent's diagnosis is the documented behaviour.
- **Git 2.54 (released in April 2026, by Collabora's 27 April account) solved
  the general problem natively**: hooks are
  declared in config as `hook.<name>.command` and `hook.<name>.event`, at
  system, global or local scope; several run for one event "in the order they
  are discovered during the config parse", and "The default <hook-name> from
  the hookdir is run last"; `hook.<name>.enabled=false` in a repository's
  uncommitted `.git/config` disables a global one there; `git hook list
  --show-scope` says where each comes from. Parallel execution follows in
  2.55, with the commit-message hooks always sequential (git-hook and
  git-config manuals, the 2.54 release notes, Collabora's account).
- **The development machine does not have it**: git 2.52.0 on 2026-10-23,
  `git hook list` unknown (probe). A hand-over that depends on 2.54 would not
  run for this user today.
- **Git's order is the reverse of this intent's criteria**: config hooks
  (the person's) run first, the `core.hooksPath` hook (the repository's)
  last. The native hand-over honours "repository check first"; Git's model
  does not.
- **The contracts the hand-over preserves** (githooks manual): `commit-msg`
  takes the message file; `pre-push` takes the remote name and location and
  reads `<local-ref> SP <local-object-name> SP <remote-ref> SP
  <remote-object-name> LF` lines on stdin; `prepare-commit-msg` "is not
  suppressed by the `--no-verify` option"; non-executable hooks are ignored; hooks run at the
  root of the working tree (`$GIT_DIR` in a bare repository), and only the
  server-side receive hooks (`pre-receive`, `update`, `post-receive`,
  `post-update`, `push-to-checkout`) run in `$GIT_DIR`, so `pre-push`, the
  push hook abcd ships, runs at the tree root like the others.
- **How the hook managers treat a person's own hooks**: pre-commit installs
  "in a migration mode which runs both your existing hooks and hooks for
  pre-commit" and seeds clones via `init.templateDir`; husky sources a
  user-level `~/.config/husky/init.sh` before every hook; lefthook merges a
  `lefthook-local.yml` meant for a person's own additions, kept out of the
  repository by a global gitignore.
- **Where a code-naming setting may live**: Git's answer is config scope: a
  global hook runs everywhere and a repository can only disable it;
  `includeIf` (`gitdir:`, `hasconfig:remote.*.url:`) lets a person scope
  personal hooks to a set of repositories from their own global file.

**Path 2**: the native hand-over as designed, shaped to Git's config-hook
model so the two converge: a setting that is a global git-config key (never
abcd's user config, never anything a repository can set), a hand-over
contract identical to `git hook run` (same arguments, stdin streamed through,
an absent personal hook ignored quietly), and `ahoy` naming `git hook list
--show-scope` as the authoritative listing on a git at or above 2.54. The
hand-over line is the seam: when 2.54 is the floor it is deleted and the
committed hooks become plain hookdir hooks that git runs last. Ship the five
events abcd uses; Git's own model covers every other event for a 2.54 user
without an empty file per event. The ordering difference against the
acceptance criteria is put to the product thinker before the hand-over is
promised as the long-term shape. The independent fit-challenge runs at plan
time.

## Open Questions

- Which events abcd ships hooks for: the five it or users commonly need (pre-commit, pre-merge-commit, prepare-commit-msg, commit-msg, pre-push) or every client-side event (post-commit, post-checkout, post-merge and so on). Shipping all of them puts empty hand-over hooks in every managed repository.
- Where the setting lives: a global git config key (for example `abcd.userHooksPath`) or abcd's own user-level config. It must never be readable from anything a repository can set, since it names code to run.
- (SOTA 2026-10-23) Whether "the repository's check runs first, the personal
  hook after" survives contact with Git 2.54's own order (config hooks first,
  the hookdir hook last), or the criterion is relaxed to "both run, either
  failure stops the act" so the hand-over can be retired when 2.54 is the
  floor.
- (SOTA 2026-10-23) What git version `ahoy` treats as the floor for naming
  `hook.*` config as the route, and what it says on an older git (2.52 on the
  development machine today).
- A missing folder or a non-executable hook: warn and carry on, or refuse? (The acceptance criteria only require that it is never silent.)
- Migration: when `ahoy` finds a global `core.hooksPath` that holds real hooks, should it offer to name that folder as the personal hooks folder?
- Repositories whose hooks are installed in `.git/hooks` instead of a committed `.githooks/`: in scope, or left as they are?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
