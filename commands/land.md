---
name: land
description: "Push a branch whose preflight passed and queue its pull request: Writes a push to origin and arms a merge; refuses the default branch or a dirty tree."
argument-hint: "[<branch>] [--no-push] [--no-enqueue]"
block: agents
user-invocable: false
---

# `/abcd:land`

Land a branch whose work is committed, in one command that can be run again
to resume: gate its head on a preflight receipt, push it to `origin`, and
queue its open pull request where the merge gate allows. It serves any
branch, a build lane's or not: a hand-built branch, a release branch, a
record-only chore. It never opens a pull request and never changes the
repository's settings or its merge queue's.

**What it does to the remote.** The push is a plain
`git push origin refs/heads/<branch>:refs/heads/<branch>` from the branch's
own checkout, with the repository's hooks running, never forced and never with
`--no-verify`, and never of the default branch. Arming is `gh pr merge <n>
--auto` with the merge queue's method, through the forge client already signed
in with the user's own identity.

## Run it

From the branch's checkout, with everything committed:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" land --json
```

`abcd land <branch>` lands a branch checked out in another worktree. Each step
reports a `status`: `done`, `skipped` (already done), `off` (a key or flag
turned it off), `stopped` (it cannot act yet: its `note` says what to do, and
running the command again resumes) or `left` (the act is a person's). A step
that prints a `command` gives the exact command to run by hand.

1. **gate**: skipped when a preflight receipt already names the branch's head,
   in any worktree, as the pre-push hook looks for one. Otherwise `make
   preflight` runs in the branch's checkout (it waits on the machine's load
   itself), its output on stderr, and the head must be unchanged and a receipt
   present after it. A repository whose make file declares no `preflight`
   target gets no push. The receipt is a convenience check, not a security
   boundary: CI re-runs every gate on the pushed commit.
2. **push** (`land.push`): the plain push above. Before it, `origin`'s fetch
   and push URLs must both be the repository the forge client resolves here.
3. **enqueue** (`land.enqueue`): the branch's one open pull request, from this
   repository (a fork's is refused), whose head on the forge is the pushed
   head. The merge rule is read from the ruleset mirror
   (`.abcd/work/rulesets/`) at the merge base with `origin`'s default branch,
   never at the branch's tip. Today every repository runs in human mode, so
   auto-merge is armed only where a merge queue gates the default branch and a
   ruleset requires a person's approval (ruling AM1); anywhere else the step
   is `left` and says why. After arming it confirms the pull request is in the
   queue, toggling auto-merge once when the forge reports it CLEAN and
   unqueued, and otherwise names what holds it. With no pull request the step
   is `stopped` with the `gh pr create` command: relay it to the user, who
   opens the pull request; the next `abcd land` queues it.

The payload carries `branch`, `head`, `checkout` (home-relative),
`default_branch`, `push` and `enqueue` (each `key`, `on`, and the `layer` and
`origin` that decided it), `pull_request` and `pull_request_url` when one was
found, and `steps`, each with `name`, `status`, `note` and, when it prints
one, `command`.

Tell the user what was pushed and where the pull request stands.

## Turning an act off

`land.push` and `land.enqueue` are booleans in `.abcd/config.json` and
`~/.abcd.noindex/config.json`, both on by default:

```json
{ "land": { "push": true, "enqueue": false } }
```

Off wins from any layer: either file can turn an act off, and neither can
turn it back on. `--no-push` and `--no-enqueue` turn one off for this run, and
no flag turns one on. A step that is off names the layer and file that turned
it off, and the push step prints the push command to run by hand.

## When it refuses

Exit 2, naming what was done before the refusal: the default branch, a
detached HEAD, uncommitted changes (staged, unstaged or untracked), a branch
checked out in no worktree, a preflight that failed or minted no receipt, a
head that moved while the preflight ran, an `origin` that is not the forge's
repository, a push git or the pre-push hook refused, more than one open pull
request, a fork's pull request, a pull request that conflicts with the default
branch, an unreadable ruleset mirror, and the forge client missing while an act
needs it. Relay the refusal; never retry it with `--no-verify`, a force, or a
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
