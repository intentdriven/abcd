# Landing checked work in private repositories without renting CI twice: state of the art

Dated 2026-10-26. A state-of-the-art pass behind the draft intent
[itd-2610261025516272](../../intents/drafts/itd-2610261025516272-an-abcd-managed-private-repository-lands.md),
local-first landing for abcd-managed repositories, asked for by the technical facilitator,
who also set the assumption the pass works under: most repositories abcd
manages are private. The question: how a managed repository should be set up
so that agents check their own work before it merges (quality), only the
checks a change needs are run (speed), and checks run on the person's own
machine, with a hosted runner used only where nothing local can stand in
(cost). Every source below was opened in the pass. Tiers: T1 is primary
documentation, source code or peer-reviewed work, T2 a credible engineering
write-up, T3 a blog or anecdote.

## What abcd's own repository measures

abcd's own repository is the one managed repository with the full regime (a
preflight, a push receipt, a merge queue), so it is the baseline. It is public,
so its hosted minutes are free, and its costs are wall-clock and queue time.

- One `make preflight` runs for about 23 to 24 minutes, and the wait at the
  load gate before it took 37 to 62 minutes; about 20 waiters were queued
  across seven sessions on 2026-10-16 (iss-2610161639025702).
- Every lane starts with a cold Go cache. The first preflight in a fresh
  worktree reused 0 cached test results out of about 110 packages; reruns in
  the same worktree reused 47 to 88 (eight preflight logs in the main
  checkout's local tier, 2026-10-24 to 2026-10-26). The cause is in the go
  command: without `-trimpath`, a package's build action ID includes its
  directory, so every worktree is a different cache key. T1,
  `cmd/go/internal/work/exec.go` line 299 in go1.26.10 ("the object file may
  refer to the absolute directory containing the package").
- Every reviewed step branch gets its own full run, and a records-only change
  pays the same Go test cost as a code change (iss-2610161639025702).
- The pull-request CI run repeats the local preflight (iss-2610110939559446).
  Over 71 merged pull requests, 2026-10-05 to 2026-10-11, the queue ran 1.27
  `merge_group` runs per merged pull request, and the macOS leg dropped queue
  entries at its 45-minute cap (iss-2609012313465609).
- Nothing reaches managed repositories. `ahoy` installs the commit-msg,
  pre-commit, pre-merge-commit and prepare-commit-msg hooks
  (`internal/core/ahoy/defaults/`), and `launch scaffold` writes the release
  workflows and reads the repository's own CI check names
  (`internal/core/launch/scaffold/cichecks.go`). Neither installs a preflight,
  a receipt, a pre-push hook or a CI check workflow.

## Findings

### What a private repository pays, and what its plan allows

- Hosted minutes included each month for private repositories: 2,000 on
  GitHub Free, 3,000 on Pro and Team, 50,000 on Enterprise Cloud. Standard
  runners cost $0.006 a minute on Linux (2-core), $0.010 on Windows and $0.062
  on macOS, so a macOS minute costs about ten Linux minutes. Public
  repositories on standard runners and all self-hosted runners are free. T1,
  [GitHub Actions billing](https://docs.github.com/en/billing/concepts/product-billing/github-actions),
  read 2026-10-26.
- A $0.002-a-minute platform fee on self-hosted runners in private
  repositories was announced on 2025-12-16 for 2026-03-01 and then postponed,
  not withdrawn. The billing page read today states self-hosted use is free.
  T2, [The Register, 2025-12-17](https://www.theregister.com/2025/12/17/github_charge_dev_own_hardware/);
  T2, [Techzine](https://www.techzine.eu/news/devops/137396/github-bends-to-criticism-and-delays-paid-self-hosting-of-runners/).
- GitHub's merge queue is available in public repositories owned by an
  organization, and in private repositories only when the owning organization
  uses Enterprise Cloud. T1,
  [Merging a pull request with a merge queue](https://docs.github.com/en/pull-requests/how-tos/merge-and-close-pull-requests/merging-a-pull-request-with-a-merge-queue).
  So for most private managed repositories the queue route of itd-115 is
  closed: a personal account or a Team organization has no queue to configure.
- Rulesets work in public repositories on GitHub Free, and in public and
  private repositories on Pro, Team and Enterprise Cloud. T1,
  [Managing rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets).
  A private repository on GitHub Free can therefore enforce nothing on the
  forge: the local hooks are the whole fence.
- A required status check can accept a status from any source, which anyone
  with write access can set, or only from a named app. T1,
  [Available rules for rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets).
  With required checks on, commits are either pushed to another branch and
  merged, or pushed directly once their checks have passed; strict checks
  (the branch up to date with its base) mean more builds, and loose checks
  mean fewer builds but checks that can fail after the merge. T1,
  [About protected branches](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches).
- Where a queue exists, it "provides the same benefits as the Require branches
  to be up to date before merging" setting without the author updating the
  branch, needs the `merge_group` trigger or the required checks never report,
  and admits a pull request only once it has passed its required checks. Its
  settings include build concurrency (1 to 100), minimum and maximum group
  size, a wait time and "only merge non-failing pull requests". T1,
  [Managing a merge queue](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue).

### Local CI that the forge can see

- gh-signoff records a local pass as a commit status (context `signoff`, or
  `signoff/<step>` for a partial sign-off) on a commit already pushed, and its
  `install` writes a ruleset requiring it. It vouches by the developer's word:
  nothing proves the run happened, and repository admins can bypass it. Its
  stated rationale: "Dev laptops are super fast these days. They're
  chronically underutilized. And you already own them." T1,
  [basecamp/gh-signoff](https://github.com/basecamp/gh-signoff).
- Rails 8.1 ships local CI as a framework default: `config/ci.rb` declares the
  steps, `bin/ci` runs them, and a last step signs off with gh-signoff. The
  HEY suite took over ten minutes in cloud CI and runs locally in 1m 23s on a
  Framework Desktop and 2m 22s on an M4 Max. T1,
  [Rails 8.1 release notes](https://rubyonrails.org/2025/10/22/rails-8-1).
- Self-hosted runners are free but carry no guarantee of an ephemeral clean
  machine, "can be persistently compromised by untrusted code in a workflow",
  and on a private repository anyone who can fork it and open a pull request
  (generally anyone with read access) can reach the runner. T1,
  [Secure use reference](https://docs.github.com/en/actions/reference/security/secure-use);
  [Self-hosted runners](https://docs.github.com/en/actions/concepts/runners/self-hosted-runners).

### Testing the merge, not the branch

- The "not rocket science rule": keep a branch that always passes its tests,
  by testing each change integrated with the branch before the branch moves,
  not after. Jane Street scaled it with speculation (test several requests in
  parallel), batching (merge a group and retest individually on a failure),
  faster incremental builds so small changes verify quickly, and fewer
  requests: feature branches reviewed in full before integration cut their
  weekly requests from hundreds to thirty or forty. Their conclusion: the
  tricks "tweak the constants, but don't change the asymptotics". T2,
  [Making "never break the build" scale](https://blog.janestreet.com/making-never-break-the-build-scale/).
  Graydon Hoare's original post could not be fetched (HTTP 504); the rule is
  cited through this write-up.
- GitHub's strict up-to-date setting is the same rule enforced per pull
  request. Where no queue exists, re-testing every armed pull request after
  each merge is the cost of keeping it (iss-2609012313465609, cycle one).

### Running only what a change can affect

- Pants selects targets from git: `--changed-since=<merge-base>
  --changed-dependents=transitive` runs every target a change can reach, the
  form it recommends for pull requests. A change to a lockfile can mark all of
  its users changed, and transitive third-party dependencies are not
  understood in this mode. T1,
  [Pants advanced target selection](https://www.pantsbuild.org/stable/docs/using-pants/advanced-target-selection).
- Nx has moved from project-graph selection to task inputs: a task is affected
  when one of its declared inputs changed, or when it reads the outputs of an
  affected task; a lockfile change affects every task unless narrowed. The
  earlier project-graph mode remains behind `affected.legacy`. T1,
  [Nx affected](https://nx.dev/docs/features/ci-features/affected).
- Meta's predictive test selection still reports over 95% of individual test
  failures and over 99.9% of faulty changes while halving the cost of testing
  changes. It is a learned model trained on a large change history, not a fit
  for a single repository's few hundred changes. T1, Machalica, Samylkin,
  Porth and Chandra, [Predictive Test Selection](https://arxiv.org/abs/1810.05286),
  camera-ready 2019.

### Reusing results whose inputs did not change

- Go caches successful package test results. A run matches when the test
  binary is the same and only cacheable flags are passed (`-run`, `-short`,
  `-timeout`, `-v` and a few others); tests that open files within the module
  or read environment variables match only while those are unchanged, and
  `-count=1` disables the cache. T1, `go help test` (go1.26.10). The cache
  sees only files opened inside the module and environment variables read by
  the test process itself: a file read by a subprocess the test runs (git, for
  example) or outside the module is invisible to it, so such a test can reuse
  a stale pass.
- `GOCACHEPROG` names an external program that implements the build cache,
  the seam for a cache shared between machines or with CI. T1,
  `go help environment` (go1.26.10).
- Content-addressed reuse and dependency selection answer the same question
  from opposite ends: selection guesses what a change can reach from the
  graph, while reuse reruns whatever's inputs differ, byte for byte. Reuse is
  exact where the inputs are visible and wrong where they are not; selection
  is approximate everywhere. Nx's move to declared task inputs, above, is the
  field moving towards reuse.

## Fit challenge (to run at planning)

The independent fit challenge that [prefer-sota](../../principles/prefer-sota.md)
requires was not run in this pass: a peer session's preflight was in its test
stage and the machine's agent rule allowed no new sub-agent. These are the
collisions to hand it:

- **Self-attested results against [evaluator-outside-the-loop](../../principles/evaluator-outside-the-loop.md).**
  The gh-signoff stance trusts the person's word. abcd's agents push under the
  person's identity, so a sign-off is the agent's word too. The challenge:
  does a deterministic test run need an independent evaluator, or only an
  honest one, and what makes an agent's run honest?
- **No new dependency.** gh-signoff, Pants, Nx, Bazel and a remote cache are
  each a dependency or a toolchain. Posting a commit status is one forge API
  call abcd already has the client for, and Go's own cache needs only a flag.
- **Host-agnostic and forge-agnostic.** The tier facts above are GitHub's.
  A route chosen from what the forge offers keeps GitHub specifics behind the
  forge seam (itd-2610161647248332 finds out what the forge can do).
- **The strict gate's invariant (iss-172).** Strict up-to-date was kept as the
  duplicate-id gate. Ids are collision-proof since itd-114 shipped, and a local
  queue that tests the merge result before the branch moves keeps the
  invariant's purpose without the forge setting. The challenge: is that a
  reversal of iss-172 or the same rule enforced elsewhere?

## What this suggests for abcd

- Reuse before selection: make results reusable across worktrees
  (`-trimpath`) and let the cache decide what reruns, declaring the tests whose
  inputs it cannot see, rather than building a dependency-graph selector first.
- One full check per landing, on the tree that lands. A feature lands as one
  change; its steps are checked with reuse.
- For a private repository without a queue, test the merge locally, one
  landing at a time, and merge exactly the tree tested: the rule's guarantee
  at no hosted cost.
- Post the local result as a commit status so a ruleset can require it where
  the plan allows rulesets, and say plainly where it does not.
- Spend hosted minutes only where a local run cannot stand in: outside
  contributions, and whichever merge-result or other-OS run the person chose,
  priced per landing before they choose. Never put macOS on every landing in a
  private repository.
