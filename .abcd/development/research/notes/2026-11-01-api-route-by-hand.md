# The API route, run by hand: what failed and what worked

Dated 2026-11-01. A spike write-up from one evening of running abcd's agent
work on the person's API key by hand, after the subscription's weekly budget
came close to its limit. Three sessions (one draining the ledger, one on the
load stream, one on CI cost) moved their agents to the claude runner while the
verbs that would do so on their own did not yet exist. Every step below was
done by hand, so each one is a part of the route abcd should automate, and
each failure is a gap it should close. The decisions it led to are
dcn-2611012102280663 (the route is a per-repository choice, the subscription
the shipped default), dcn-2611012105401437 (the terminal drive keeps its
subscription default) and dcn-2611012113432054 (a runner that fails to start
pauses the lane for its key rather than falling back to the subscription).

## Reaching a key

- **The runner cannot use the subscription login.** `claude --bare` reads no
  OAuth login. With no key in its environment it reports `Not logged in ·
  Please run /login`, and it reports it as the `result` line of its
  stream-json output on stdout; stderr stays empty. Any remedy that carries
  only stderr misses it (iss-2611012049508108).
- **The loop records only an exit status.** A routed implementer that failed
  for that reason left `2.1.316 exited with status 1` in the run's fallbacks,
  with no word of why. Only a hand run of the same command showed the cause
  (iss-2611012049508108).
- **The key lives in the login keychain, read per command.** The working
  recipe is two statements: read the generic-password item into a shell
  variable, then pass it as a one-command environment prefix to the abcd
  call, so only that process and the claude it starts see it. It is never
  exported, written to a file or put in a setting.
- **The guard refuses the one-line form.** A command substitution inside a
  double-quoted environment prefix, and a command string run from a variable
  set by a substitution, are both refused by abcd's guard as uninspectable.
  The recipe has to stay two statements.
- **A key in a store the command line cannot list is invisible.** The second
  key was first saved where the `security` tool could not see it (the
  iCloud or Local Items store, or another account). Copying it into the login
  keychain as a generic-password item under its own service name made it
  usable. Nothing in abcd says where it looks for a key or how many it found
  (iss-2611012054262885).
- **Routing needs a hand-written roles block.** Every role
  (`implementer`, both reviewers, `intent-auditor`, `drain-judge`,
  `planning-interviewer`) had to be listed by hand under `roles` in the
  machine configuration, and the person could not tell what the block was for
  (iss-2611012054262885).

## The loop and its fallbacks

- **The first routed step decides the lane.** A routed step run without the
  key failed to start the runner, the role fell back to the host, and that
  fallback was final: a later `--route` is refused because the step hands no
  role out, `--restart` hands the role to the host again even with the route
  now in the machine configuration, and the lane cannot be discarded because
  it is not held (iss-2611012047534449). Two of Eve's lanes and one of this
  session's are parked for that reason. Until the fix lands, pass the key on
  the very first routed step.
- **A failed start should pause, not spend the subscription.** The person
  ruled that a runner failing to start pauses the lane for its key and retries
  once per later step (dcn-2611012113432054), instead of the silent fallback
  to the session's subscription that parked the lanes above.
- **zsh does not split an unquoted variable.** Three `--route` flags kept in
  one shell variable reached abcd as a single argument (`unknown flag:
  --route implementer`). A driver must pass each flag as its own word.

## Review pairs outside a loop

- **Only the loop's own validators run on the runner.** A records or docs
  branch, and a loop's records commit that is not mark-only, need a review
  pair for `abcd land --record-review` under facilitator.mode auto, and
  nothing runs that pair on the runner. Host subagents bill the subscription
  (iss-2611012054270386).
- **The stand-in: the reviewer's own definition, run on the key.** Running
  `claude --bare -p` with the reviewer's agent file as an appended system
  prompt produced usable returns, each ending in a `### Verdict` naming the
  head.
- **`--bare` gives an agent only Read, Edit and Bash.** It has no Grep or
  Glob, and a narrow Bash allowance (`Bash(git:*)`) under `dontAsk` still
  left one reviewer unable to open the diff, so it returned no verdict.
- **Passing the diff on stdin works.** The reliable form was a reviewer with
  no tools at all, handed the full `git log -p --stat` of the range, the
  rules it judges against and the cited code, plus a list of the file each
  cited record id resolves to, so it does not reach for a tool to check them.
- **A tool-less reviewer must be told it has no tools.** Twice a reviewer
  with no tools wrote tool-call markup instead of a verdict and stopped. An
  explicit line saying it has no tools and must not write tool calls fixed it.
- **`claude -p` waits three seconds for stdin.** Without stdin redirected it
  warns `no stdin data received in 3s` and goes on; redirecting from
  `/dev/null` or from the diff silences it.

## Cost and churn

- **Every amendment restarts the head-bound checks.** A review receipt and a
  preflight receipt each name one head. Each fix to the records branch moved
  the head, so both reviews and the preflight ran again in full, six times in
  one evening, though most fixes changed one sentence. A delta review over the
  last reviewed head would have cost a fraction of that (iss-2611012123291272).
- **Driving still bills the subscription.** Even with every handed-out agent
  on the key, the session that drives the loop, reads the returns and writes
  the records spends the subscription. The terminal drive
  (itd-2610222215485265) is the route that takes the driving off it, and it
  keeps its subscription default by the person's ruling.
- **Subagents a session starts itself bill the subscription.** The host's own
  agent tool has no way to send its agents to the key, so the question drafter
  and any hand-started review ran on the subscription.

## What to automate, in order

1. Read the key from named keychain services, with two keys and a spending
   cap (Henry's key-supply work, itd-2611011230226453).
2. A failed runner start pauses the lane and carries the runner's own report
   (iss-2611012047534449, iss-2611012049508108).
3. A verb that runs a review pair on the runner over a named head, handing it
   the diff and the resolved ids rather than relying on tools
   (iss-2611012054270386).
4. Offer the per-repository route choice at install, with the subscription as
   default (iss-2611012054262885).
5. A delta review for a head that moved by records alone (iss-2611012123291272).
