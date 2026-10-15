---
name: land
description: "Push a branch whose preflight passed and queue its pull request: Writes a push to origin and arms a merge; refuses the default branch or a dirty tree."
argument-hint: "[<branch>] [--no-push] [--no-enqueue]"
block: people
---

# `/abcd:land`

Land a branch whose work is committed, in one command that can be run again
to resume: gate its head on a preflight receipt, push it to `origin`, and
queue its open pull request where the merge gate allows. It serves any
branch, a build lane's or not: a hand-built branch, a release branch, a
record-only chore. It never opens a pull request and never changes the
repository's settings or its merge queue's.

**It runs the branch's own code.** The gate is the branch's own `make
preflight`, run with the user's environment and credentials, so run land only
on a branch whose content the user trusts; a branch from someone else is
reviewed before it is landed.

**What it does to the remote.** The push is a plain
`git push origin <head>:refs/heads/<branch>` of the commit the gate checked,
from the branch's own checkout, with the repository's hooks running, never
forced and never with `--no-verify`, and never of the default branch. Arming
is `gh pr merge <n> --auto` with the merge queue's method and
`--match-head-commit=<head>`, through the forge client already signed in with
the user's own identity.

## Run it

From the branch's checkout, with everything committed:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" land --json
```

`abcd land <branch>` lands a branch checked out in another worktree. Each step
reports a `status`: `done`, `skipped` (already done), `off` (a key or flag
turned it off), `stopped` (it cannot act yet: its `note` says what to do, and
running the command again resumes), `left` (the act is a person's) or
`landed` (the head is already on the default branch). A step that prints a
`command` gives the exact command to run by hand. With `land.push` and
`land.enqueue` both off nothing runs, the gate included. With `land.push` off
and `land.enqueue` on, the gate still runs: arming a pull request says its head
is ready to merge, and the passing preflight is what makes it ready.

Before any step, `origin`'s every fetch and push URL must be the repository
the forge client resolves here (an SSH host alias is refused, naming it, with
the `insteadOf` rule that maps it in the URL's own form: land does not read ssh
config; a URL on another real host is refused as on a different host, with the
`git remote set-url` that points `origin` back), and the forge's own default
branch must be the one `origin/HEAD` names here and not the branch. Then
`origin`'s default branch is fetched, and the head has landed when it is on
it, or when the forge lists a merged pull request from this repository's
branch whose head was this head and whose base was the default branch (a
squash or rebase merge puts another commit on the default branch). Either way the one step reported is `land` with status
`landed` (naming the pull request when the forge proved it), and nothing is
gated, pushed or queued, so a re-run after the merge never recreates the
deleted branch, whichever merge method the queue used. A pull request merged
at another head, from a fork, or into another base (a stacked pull request)
proves nothing about this one, and a forge that cannot be asked gets no gate
and no push; nor does a listing of 20 merged pull requests from branches of
that name with none at this head, since the proof may lie past the bound.

1. **gate**: skipped when a preflight receipt already names the branch's head,
   in any worktree, as the pre-push hook looks for one. Otherwise `make
   preflight` runs in the branch's checkout (it waits on the machine's load
   itself), its output on stderr, and the head must be unchanged and a receipt
   present after it, and `origin`'s URLs exactly as they were checked (the
   branch's own preflight could add a push URL or a `pushInsteadOf`). A
   repository whose make file declares no `preflight` target gets no push. The receipt is a convenience check, not a security
   boundary: CI re-runs every gate on the pushed commit.
2. **push** (`land.push`): the plain push above, of the gated commit whatever
   the branch names by then. A push git refuses carries git's own per-ref
   report.
3. **enqueue** (`land.enqueue`): the branch's one open pull request from this
   repository (a fork's pull request from a branch of the same name is set
   aside, and refused only when it is the only one), whose head on the forge
   is the pushed head. The merge rule is read from the ruleset mirror
   (`.abcd/work/rulesets/`) at `origin`'s default branch as just fetched,
   never at the branch's tip. Today every repository runs in human mode, so
   auto-merge is armed only where a merge queue gates the default branch and a
   ruleset requires a person's approval (ruling AM1); anywhere else the step
   is `left` and says why. A pull request someone else already armed is left
   as it is and its queue state reported. After arming it confirms the pull
   request is in the queue at the head it armed, toggling auto-merge once when
   the forge reports it CLEAN and unqueued, and otherwise names what holds it;
   a toggle whose re-arming fails says the pull request is now disarmed and
   gives the command that arms it again. A re-run never toggles an arming
   again, its own included, so an armed pull request still CLEAN and outside
   the queue is reported with what the user can do: wait for the forge to
   queue it, or withdraw and re-arm it with the exact `gh pr merge` commands
   the note gives. With no pull request the step is
   `stopped` with the `gh pr create` command: relay it to the user, who opens
   the pull request; the next `abcd land` queues it.

The payload carries `branch`, `head`, `checkout` (home-relative),
`default_branch`, `push` and `enqueue` (each `key`, `on`, and the `layer` and
`origin` that decided it), `pull_request` and `pull_request_url` when one was
found (or proved the head merged), `pushed` (the commit the push sent) when it
pushed, `landed` when the head had already landed, and `steps`, each with `name`, `status`, `note` and, when it prints
one, `command`.

Tell the user what was pushed and where the pull request stands.

## Turning an act off

`land.push` and `land.enqueue` are booleans in `.abcd/config.json` (read in
the working directory's checkout and in the landed branch's own) and
`~/.abcd.noindex/config.json`, both on by default:

```json
{ "land": { "push": true, "enqueue": false } }
```

Off wins from any layer: any of the files can turn an act off, and none can
turn it back on. When the home directory is refused (it lies inside a git
working tree that is not the account's), the machine's file cannot be read,
so both acts are off and say why. `--no-push` and `--no-enqueue` turn one off for this run, and
no flag turns one on. A step that is off names the layer and file that turned
it off, and the push step prints the push command to run by hand.

## When it refuses

Exit 2, naming what was done before the refusal: the default branch, a
detached HEAD, uncommitted changes (staged, unstaged or untracked), a branch
checked out in no worktree, a preflight that failed or minted no receipt, a
head that moved while the preflight ran, `origin`'s URLs changed while the
preflight ran, an `origin` URL that is not the forge's repository, a forge
default branch that differs from the local one or is the branch, a forge that
cannot be asked (whether the branch already merged included), a merged
listing full with none at the head, a push git or the pre-push hook
refused, more than one open pull request from this repository, only a fork's
pull request, a pull request that conflicts with the default branch, a head
that moved on the forge after arming, an unreadable ruleset mirror, and the
forge client missing while an act needs it. Relay the refusal; never retry it with `--no-verify`, a force, or a
hand-run `gh pr merge` the merge gate did not allow.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
