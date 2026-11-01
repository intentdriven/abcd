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
- **The fallback is silent at the surface.** The step that fell back exited
  0, about two seconds after the await was posted, and said only that the
  host now runs the implementer; the reason (`2.1.316 exited with status 1`)
  sat in the JSON. Nothing looked wrong until the JSON was read.
- **A fallen-back lane still takes a slot.** On a run whose ceiling is two
  lanes, a `--route` after the fallback was refused because the step would
  have opened a third lane for the next spec step, not retried the parked
  one: the parked lane counted as out.
- **A lane can be handed to the host before any route exists.** One
  `implement step` finished a lane and opened the next through its brief and
  implement stages with no pause between, so the second lane was told to the
  host before a route could be set for it.
- **An elapsed window refuses a route.** A run whose 120-minute window
  lapsed while its preflight queued refused `--route` because the step would
  only pause; stepping once without it took the zero-minute pause, and the
  next step took the route (iss-2611012124359297).
- **The planning interview on the key ended on a refused question.** Run from
  a plain terminal, the routed interviewer wrote a sound question within a
  minute, abcd's question check refused it for its rows (39 against the
  terminal's 32), the refusal counted as the runner's invalid answer with no
  host to fall back to, and the interview ended on an internal message
  (iss-2611011402496418, dcn-2611011402510346, dcn-2611012123240576). The
  fallback record kept the reason but not the cost, so a refused launch's
  spend was lost.
- **zsh does not split an unquoted variable.** Three `--route` flags kept in
  one shell variable reached abcd as a single argument (`unknown flag:
  --route implementer`). A driver must pass each flag as its own word.

## Landing by hand

- **A reviewed follow-up moves the branch past the head the landing lands.**
  The loop refused a lane whose branch carried a reviewed follow-up commit
  (`… is at <sha>, not at the head <sha> the landing lands`), and the lane was
  landed by hand with `abcd land` and `gh pr create` instead
  (iss-2611011943357182).
- **A lane landed by hand stays stuck after its merge.** Once that pull
  request merged, `implement step` still refused at land on the head
  mismatch, though both heads were on the default branch; moving the branch
  back as the remedy asked led to a demand for a full preflight of the
  already-merged records commit, one to two hours on this machine. The loop
  should see that its landing head is already on the default branch before
  its head and receipt checks (iss-2611011943357182).
- **`land --record-review` takes one return per flag.** Two paths after one
  flag read the second as the branch (`the branch's head cannot be resolved`).
- **A return must name the head, and a reviewer's own format can drop it.**
  `land --record-review` refuses a return that does not contain the head's
  first nine characters. A security reviewer run from its own definition
  twice left the sha out of its verdict line though the prompt asked for it;
  an output contract appended to the system prompt, making the first line
  `Reviewed head: <sha>`, fixed it.
- **The verdict word must open the verdict section.** `land --record-review`
  takes a return only when the first non-empty line under `### Verdict` is the
  verdict word; a first line such as `Head: <sha>` or `Verdict: SHIP` is
  refused as not one a validator writes (iss-2611012124381670).
- **The loop's own reviews do not cover its landing commit.** The landing's
  resolve commit lands after the reviewed head, and `abcd land` wants a fresh
  pair naming it, which cost two extra subscription reviews for a records-only
  commit (iss-2610311732460169, iss-2610312044511316).
- **A return keeping an earlier round's verdict is refused.** A return with
  two `Verdict` headings at any level is refused (`it has 2 Verdict sections`),
  so a reviewer that keeps earlier rounds must rename their headings.
- **The away queue needs an away record.** `abcd mode queue` refuses when the
  person is not away, so a finding meant for their queue while they are
  present has to be put to them directly.

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

- **Measured cost of one review round on the key.** In one loop run's first
  validate round, the ruthless reviewer cost 0.9654 USD and the security
  reviewer 0.3326 USD, both on claude-opus-5-5, the runner's own default with
  no model set, and no rate limit was hit; the token counts are in the run's
  route receipts. The record of a failed launch drops its cost, as the claude
  runner drops `total_cost_usd` on a non-success result.
- **Permission modes differ by role.** A role granted the shell (the loop's
  reviewers) launches on the shell route, with the guard answering its
  permission prompts, even when it runs no command; a role without it (the
  planning interviewer) launches under `dontAsk`. Both launched under
  `--bare` from inside a host session.
- **The person ruled on key handling and caps.** One named key per session,
  read from the operating system's secret store; caps per day, week and month
  that pause and wait at the limit; a run with no cap set refuses to start;
  and a cap is raised only by a command refused to agents
  (dcn-2611012123470476, itd-2611012123486967).

- **The load gate, not the agents, set the pace.** Code preflights waited 1
  to 2.5 hours each at the load gate in the afternoon, with seven or more
  waiters behind one that had run more than two hours, and the load peaked
  near 290 on 16 cores by a peer's count; records-only preflights took the
  fast path in a few minutes. The machine ran out of application memory once
  that evening (iss-2611011737317454).
- **The merge queue's macOS race pass hit its watchdog.** It was stopped at
  1h20m and dropped two pull requests from the queue (evidence on
  iss-2610181757540736).
- **A re-review of a small delta took 1 to 4 minutes per reviewer.** That is
  the cost a delta review would pay each round, against a full pair.
- **The installed release judges questions at 24 rows while main's default is
  32.** A question drafted against main's figure gets the installed hook's
  rows note.

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
