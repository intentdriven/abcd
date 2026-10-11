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

## Open Questions

- Which events abcd ships hooks for: the five it or users commonly need (pre-commit, pre-merge-commit, prepare-commit-msg, commit-msg, pre-push) or every client-side event (post-commit, post-checkout, post-merge and so on). Shipping all of them puts empty hand-over hooks in every managed repository.
- Where the setting lives: a global git config key (for example `abcd.userHooksPath`) or abcd's own user-level config. It must never be readable from anything a repository can set, since it names code to run.
- A missing folder or a non-executable hook: warn and carry on, or refuse? (The acceptance criteria only require that it is never silent.)
- Migration: when `ahoy` finds a global `core.hooksPath` that holds real hooks, should it offer to name that folder as the personal hooks folder?
- Repositories whose hooks are installed in `.git/hooks` instead of a committed `.githooks/`: in scope, or left as they are?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
