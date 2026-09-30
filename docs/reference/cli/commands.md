# CLI command reference

This page is generated from the abcd command tree by `GenerateReference` in
`internal/surface/cli`. It is a derived artefact: do not edit it by hand. A
drift test regenerates the tree and fails the build whenever this page and the
tree disagree, so the reference can never silently go stale. Regenerate it with
`go generate ./internal/surface/cli`.

Every user-facing command is listed with its sentence (what it does, what it
writes, and when it refuses), its usage line, and its flags; the
operator-internal hook entrypoints, and the old spellings of moved commands,
are omitted.

## `abcd`

Render the status board, or say what one record id is and its next move: Writes nothing; refuses any other positional argument.

**Usage:** `abcd [<record-id>] [flags]`

Agent-based configuration for development.

Bare `abcd` renders the read-only status board — what can I do. A single
positional matching a record id (`iss-N`, `itd-N`, `spc-N`, `adr-N`, `adm-N`,
`srp-N` or `rfm-N`) instead reports what that record is, where it lives, and
the next move for its lifecycle state — what is this. N is either a short
ordinal from before ids were minted or the sixteen-digit stamp minted since;
both resolve. The bare and the id form are strictly read-only; any other
positional is refused as an unknown command.

**Flags:**

```
      --agent      with --help, list the verbs agents and hosts call as well, each naming the page to read next
      --json       emit machine-readable JSON on stdout; a refusal is a {"abcd":"error","error":…,"exit_code":…} object on stdout too, and exits non-zero
      --no-color   render the banner without color
      --version    print abcd's version, install mode, and vintage, from disk alone (the release check is: abcd update --check)
```

### `abcd ahoy`

Detect abcd's install state and list its gaps, or report one mode a flag names: Writes nothing; refuses any argument or two modes at once.

**Usage:** `abcd ahoy [flags]`

**Flags:**

```
      --dry-run     print the detection result as its JSON envelope, whether or not --json is passed
      --identity    check git's commit author and committer against .abcd/config/identity.json, exiting non-zero when either diverges (for a pre-commit hook or CI)
      --providers   explain the optional OpenAI-compatible provider adapter, list the providers configured on this machine and where a key can live
      --remote      report this repository's GitHub secret-scanning settings and what the remote apply sub-verb would change
```

#### `abcd ahoy connect`

Verify a model provider with one call, then configure it: Writes its block under ~/.abcd/ and its key to the home chosen; refuses a key typed at a terminal.

**Usage:** `abcd ahoy connect <provider> [flags]`

**Flags:**

```
      --base-url string     the provider's OpenAI-compatible base URL: https, or http to a server on this machine
      --env string          for --home external: the environment variable that holds the value
      --field string        for --home external: the dotted field of --file that holds the value (auth.token)
      --file string         for --home external: a tool's JSON configuration file under the home directory, written from ~/
      --home string         where the key lives: external (--env, or --file and --field) | abcd (read from stdin into the owner-only ~/.abcd/credentials.json) | keychain (read from stdin into the platform keychain) | none (a server that takes no key)
      --key string          the credential's name (default: the provider's name)
      --model stringArray   a model the provider may serve, repeated for each (the first allowlist; the verification call asks for the first)
```

**Example:**

```
abcd ahoy connect local --base-url http://127.0.0.1:8080/v1 --model example-model --home none
```

#### `abcd ahoy credential`

List the credentials abcd reads, explain one, or verify and store it: Writes the chosen home only with --home; refuses a value the adapter's call fails.

**Usage:** `abcd ahoy credential [<name>] [flags]`

**Flags:**

```
      --env string     for --home external: the environment variable that holds the value
      --field string   for --home external: the dotted field of --file that holds the value (auth.token)
      --file string    for --home external: a tool's JSON configuration file under the home directory, written from ~/
      --home string    where the credential lives: external (--env, or --file and --field) | abcd (read from stdin into the owner-only ~/.abcd/credentials.json) | keychain (read from stdin into the platform keychain)
```

#### `abcd ahoy doctor`

Report every install gap, user-scope state included: Writes nothing; refuses any argument.

**Usage:** `abcd ahoy doctor`

#### `abcd ahoy install`

Apply the install gaps the detection finds: Writes .abcd/, the name-guard hooks and the PATH entry, and installs a tool only on a yes; refuses a stale binary.

**Usage:** `abcd ahoy install [flags]`

**Flags:**

```
      --adopt                   adopt an unmanaged repo without prompting
      --allow-stale-binary      proceed even when the running binary is stale against its source tip or its vintage cannot be determined; the default is to refuse before any write and name the rebuild fix
      --attribution             opt this repo into the committed prepare-commit-msg prompt asking every commit to declare whether a tool assisted it; the choice is recorded, so a later install without the flag keeps the hook
      --bin-dir string          directory for the PATH entry (default ~/.local/bin, or an existing abcd install adopted in place); fails when it is not writable — abcd never escalates privileges
      --dev                     track-latest dogfood mode: the PATH entry rebuilds from the source tip on every call instead of pinning the built binary
      --docs-target string      which conventions file carries the managed block, which names abcd: claude_md | agents_md | both | skip (default skip)
      --install-tool strings    answer yes to installing this missing tool (repeatable): the answer a host's question tool relays; without it a tool is installed only on an answer typed at a terminal, never on the approve-everything flag, a piped answer or CI
      --oracle-backend string   oracle backend: host-delegated | native | cli | api | mcp
      --refuse-adopt            decline to adopt an unmanaged repo
      --scan-deep string        enable deep scan: true | false
      --visibility string       repo visibility: private | public
      --yes                     approve every resolvable change category without prompting; excludes the optional git-identity pin, the status line and the model-tier routing tables, which need an answered prompt (run without --yes, or answer every prompt with: yes | abcd ahoy install); it never changes the repository's git identity, which is proposed only to a person at a terminal
```

#### `abcd ahoy remote`

Enable GitHub secret scanning and push protection: Writes nothing bare, only the settings and their mirror; refuses bare, naming `abcd ahoy --remote`.

**Usage:** `abcd ahoy remote [command]` (the bare form's work is `abcd ahoy --remote`)

##### `abcd ahoy remote apply`

Enable GitHub secret scanning and push protection on this repository: Writes both settings and their mirror; refuses an unconfirmed run.

**Usage:** `abcd ahoy remote apply [flags]`

**Flags:**

```
      --yes   confirm the remote change without being asked; without it an unanswered run declines and changes nothing
```

#### `abcd ahoy uninstall`

Remove abcd from this repository, leaving .abcd/ in place: Writes the removal of the marker block, PATH copy, and provenance record; refuses any argument.

**Usage:** `abcd ahoy uninstall [flags]`

**Flags:**

```
      --bin-dir string   directory holding the PATH entry to remove; needed only when it was installed with --bin-dir into a directory that is not on PATH
```

### `abcd banlist`

Render both banned-names layers: Writes nothing; refuses an unknown word without echoing it.

**Usage:** `abcd banlist`

#### `abcd banlist add`

Add one banned-name entry to the layer a flag names: Writes that layer's store; refuses without exactly one of --private or --public.

**Usage:** `abcd banlist add --private|--public <key> <pattern|-> [flags]`

**Flags:**

```
      --private            the gitignored per-machine layer (.abcd/.work.local/private-names.txt)
      --public             the committed, CI-enforced layer (.abcd/docs-lint.json)
      --severity string    public entry severity: blocker (default) | warn
      --successor string   public entry's replacement, cited in the finding (default "a generic term")
```

**Example:**

```
abcd banlist add --private acme-internal 'acme-internal\.example\.com'
```

#### `abcd banlist list`

Render the banned-names layers, private entries by key only: Writes nothing; refuses --private and --public together.

**Usage:** `abcd banlist list [--private | --public] [flags]`

**Flags:**

```
      --private   the gitignored per-machine layer (.abcd/.work.local/private-names.txt)
      --public    the committed, CI-enforced layer (.abcd/docs-lint.json)
```

#### `abcd banlist migrate`

Key a legacy private store in place, every line matching what it matched: Writes the private store; refuses when no private store exists.

**Usage:** `abcd banlist migrate`

Convert a legacy private store (.abcd/.work.local/private-names.txt with no
'# abcd-banlist: keyed' first line, every line a whole-line pattern) to the keyed
format: the declaration becomes line 1, and each pattern keeps its exact bytes under
the key the guard already names it by, entry-<its line>. Comments and blank lines
survive. add, remove and `abcd source sync-banlist` refuse a legacy store with entries
until it is migrated. A keyed store is left alone. No pattern is printed.

#### `abcd banlist remove`

Remove one banned-name entry from the layer a flag names: Writes that layer's store; refuses a public entry curated by hand.

**Usage:** `abcd banlist remove --private|--public <key> [flags]`

**Flags:**

```
      --private   the gitignored per-machine layer (.abcd/.work.local/private-names.txt)
      --public    the committed, CI-enforced layer (.abcd/docs-lint.json)
```

**Example:**

```
abcd banlist remove --private acme-internal
```

### `abcd build`

Start the loop that takes one READY intent to delivered: Writes the run's state file in the local tier; refuses an open question, a hold or a peer holding it.

**Usage:** `abcd build <itd-N> [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [flags]`

Start the implement loop for one intent, or resume the run already in progress for it.
A new run's checks run first, and every one must pass:
the intent is READY (planned, criteria written, its spec linked and written), asks no
open question, has no unanswered claim section, is not held, names no unshipped intent
in `blocked_by`, its spec leaves a step to build, and no peer holds it (no sibling
worktree or local branch holds it in another bucket, and no session holds a live claim
on it; a peer or claim that cannot be read counts as holding it). A refusal names the
check, the reason and the remedy, and writes nothing. `abcd build next` picks the intent
instead of taking one named.

When the checks pass, the run is created in this checkout's local tier,
`.abcd/.work.local/run/<run-id>/state.json`: one lane for the spec's first unlanded step,
the other unlanded steps pending, and the run record's first line. The tier itself is
never created: only a repository abcd manages has one. Starting again while the run is
in progress creates nothing and names the run without judging the checks again (the
run's own lanes change what they read), so a killed process resumes where it stopped.

--session names the host session's id in the shared run state (`abcd implement join`):
a new run then claims the intent there for that session, the run id as its lane, so a
build of the same intent from any other checkout of the repository is refused as held
before this run's lane has moved or claimed anything, and the session's own claim on the
intent is not counted as a peer's. A session that has not joined is refused. Without it
the run holds no claim, and the result says so.

A new run is paced: a working window, a pause after it, and a ceiling on the run's lanes
and validators alive at once. The three numbers are read once, when the run starts:
--pace <work-minutes>/<pause-minutes> and --sub-agents <n> for this run, else pace.work_minutes,
pace.pause_minutes and pace.sub_agents in the repository's .abcd/config.json, else in
~/.abcd/config.json, else the bundled 120/300 with 2 sub-agents. The result and the run
record name each number's layer. A malformed pace or ceiling, typed or configured, is
refused naming the value and the accepted form, and writes nothing. Starting again keeps
the run's pace; a flag naming another is refused. The window and the pause bind through
`abcd implement step`; the ceiling is recorded with the run, and this build does not
count lanes against it.

The run then moves one step per `abcd implement step`, driven by the host session.

Exit 2 on a refusal, exit 3 when a peer holds the intent or the run state is locked
(back off and take other work).

**Flags:**

```
      --pace string         this run's working window and pause, <work-minutes>/<pause-minutes> (e.g. 90/240); wins over every configured layer
      --session string      the host session's id in the shared run state; a new run claims the intent for it
      --sub-agents string   this run's ceiling on lanes and validators alive at once, a whole number; wins over every configured layer
```

**Example:**

```
abcd build itd-2609010000000001
```

#### `abcd build next`

Pick the readiest planned intent and start its run: Writes the run's state and the reason as the lane's first commit; refuses when nothing passes the checks.

**Usage:** `abcd build next [--session <id>] [--pace <work-minutes>/<pause-minutes>] [--sub-agents <n>] [--max <n>] [--until-empty] [flags]`

Pick the readiest planned intent, write down why, and start its run.

The candidates are the planned intents that pass every check `abcd build <itd-N>` runs
(READY, no open question, no unanswered claim section, not held, no unshipped intent in
`blocked_by`, a step left to build, no peer holding it), less one this checkout already has
a run in progress for. Each is scored from its record, three parts at equal weight, each 0
to 100: criteria clarity (the share of its acceptance criteria in Given-When-Then form), a
test path (its spec's `## Footprint` names tests) and the expected footprint (100 divided by
the packages that section names). A part whose section is absent reads zero, and the
reason says the spec carries no footprint. The readiest is taken; the oldest among equals,
and the reason then says the tie was broken by age.

The pick starts the run `abcd build <itd-N>` would start for that intent, with the pick in
the run's state. The reason is one `pursued:` grounds entry opening `picked by run <run-id>
on <date>`: every candidate with its score, the rule, the runner-up and why it lost, and the
falsifier. The lane's worktree step appends it to the intent in the lane's own worktree and
commits it there as the lane branch's first commit, record-only, before the brief; the
receipt verifier does not count that commit as the implementer's. The checkout you run this
in is never written but for the run state. `abcd intent ready` keeps reporting the person's
entry as the most recent conjecture.

One pick per invocation. --max <n> above 1 and --until-empty, which continue under the pace
rule, are refused: that half of the verb is not built in this abcd. --session, --pace and
--sub-agents are `abcd build`'s own.

No candidate is refused, naming each excluded intent and the check that excluded it, and
nothing is written. Exit 2 on a refusal, exit 3 when the chosen intent's run is already in
progress or the run state is locked.

**Flags:**

```
      --max int             how many picks to make; only 1 is built, and more is refused
      --pace string         the new run's working window and pause, <work-minutes>/<pause-minutes>; wins over every configured layer
      --session string      the host session's id in the shared run state; the new run claims the picked intent for it
      --sub-agents string   the new run's ceiling on lanes and validators alive at once; wins over every configured layer
      --until-empty         pick until no candidate is left; not built, and refused
```

### `abcd capture`

File an issue from quoted text, or render the ledger's status bare: Writes one record under open/; refuses a missing --remedy, a lone word or no checkout.

**Usage:** `abcd capture [text] [flags]`

**Flags:**

```
      --blocked-by string            comma-separated iss-N ids this issue is blocked by; each must exist in the ledger — blocked_by is documented in .abcd/work/issues/README.md under "Derived priority" and in commands/capture.md under "Link"
      --category string              issue category: bug | documentation | drift | inconsistency | tech-debt | security | ux | process | architectural-insight | future-work-seed | observation | lapse (default observation)
      --found-at string              optional repo-relative path, which must exist in this checkout, or a conceptual location in words
      --found-during string          session/command context (default manual-capture)
      --lapsed-at string             RFC 3339 instant a discipline gave way (the lapse, not the write-up)
      --production-mode string       how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
      --remedy abcd capture remedy   the proposed fix, one line (required); abcd capture remedy rewrites it later
      --severity string              severity: nitpick | minor | major | critical (default minor)
      --slug string                  override the slug derived from the text
      --source string                surfacing channel: plan-review | impl-review | manual-test | review-followup | agent-finding | agent-observation | user-observation | drift-detection | memory-curation | managed-repo (default user-observation)
```

#### `abcd capture admit`

Admit one widening proposal into its run's candidate set: Writes its accepted disposition and an adm-N record; refuses before a committed comparative run.

**Usage:** `abcd capture admit <rdi-N> --grounds "<why>" [flags]`

**Flags:**

```
      --grounds string   why the proposal is admitted (free text, held to the grounds floor; on a standing acceptance it must be that acceptance's ground)
```

**Example:**

```
abcd capture admit rdi-2609010000000001 --grounds "the widened configuration is one the next release has to serve"
```

#### `abcd capture defer`

Carry an open major or critical issue past one release cut: Writes deferred_after and deferral_reason; refuses a minor or nitpick issue, or an empty reason.

**Usage:** `abcd capture defer <iss-N> --after <vX.Y.Z> --reason <text> [flags]`

**Flags:**

```
      --after string    the current anchor: the newest vX.Y.Z release tag, which the cut measures from (required)
      --reason string   why the finding is carried past this cut rather than fixed (required)
```

**Example:**

```
abcd capture defer iss-2609010000000001 --after v0.1.0 --reason "the fix needs the parser rewrite that lands next cycle"
```

#### `abcd capture disposition`

Answer one reading item with a disposition record: Writes the record keyed to the item; refuses a second answer without --supersedes.

**Usage:** `abcd capture disposition <rdi-N> --state <accepted|rejected|declined|held> (--grounds <text>, or --exit-condition <text> when held) [--supersedes <dsp-N>] [--recurs <rdi-N,...>] [flags]`

**Flags:**

```
      --exit-condition string        what would end a held disposition (required on held; a hold exits only through a superseding disposition that cites it)
      --grounds string               disposition_grounds: why this answer (free text; required on every state except held)
      --hold-frame-location string   RESERVED (dormant): the frame element a hold sits at; a populated value is refused until activation is ruled
      --hold-moscow string           RESERVED (dormant): must | should | could | wont; a populated value is refused until activation is ruled
      --recurs string                comma-separated prior rdi-ids this item recurs from — the researcher's confirmed recognition; the ingest's duplicates/refines link is only a proposal
      --state string                 the answer: accepted | rejected | declined | held (availability varies by the item's position)
      --supersedes string            the standing dsp-N this answer replaces; required once an item already carries one
```

**Example:**

```
abcd capture disposition rdi-2609010000000001 --state accepted --grounds "pursued: the tension is real and the next reading will show it again"
```

#### `abcd capture link`

Add or remove blocked_by edges on an issue: Writes the issue's blocked_by list; refuses an id the ledger does not hold.

**Usage:** `abcd capture link <iss-N> [--blocked-by <iss-M,...>] [--unblock <iss-M,...>] [flags]`

**Flags:**

```
      --blocked-by string   append: comma-separated iss-N ids this issue is blocked by; each must exist in the ledger — blocked_by is documented in .abcd/work/issues/README.md under "Derived priority" and in commands/capture.md under "Link"
      --unblock string      remove: comma-separated iss-N ids to drop from blocked_by; each must currently be in the list. With --blocked-by in the same call the removals are applied first, then the additions
```

**Example:**

```
abcd capture link iss-2609010000000001 --blocked-by iss-2609010000000002
```

#### `abcd capture list`

List the issues in one status folder or all three: Writes nothing; refuses when no status flag is given.

**Usage:** `abcd capture list [flags]`

**Flags:**

```
      --all        issues across all three states
      --open       issues currently in open/
      --resolved   issues currently in resolved/
      --wontfix    issues currently in wontfix/
```

#### `abcd capture mentions`

List open issues that default-branch history names with no resolution behind them: Writes nothing; refuses outside a git checkout.

**Usage:** `abcd capture mentions [--ref <branch>] [flags]`

**Flags:**

```
      --ref string   history to walk (default: the repository's default branch)
```

#### `abcd capture migrate`

Rewrite retired promote back-links as related_intents and related_issues: Writes the records only with --apply; refuses outside a git checkout.

**Usage:** `abcd capture migrate [--apply] [flags]`

**Flags:**

```
      --apply   write the rewritten records (default: report only)
```

#### `abcd capture promote`

Graduate an issue or an accepted reading item into an intent draft: Writes the draft and both back-links; refuses a promoted issue or an unaccepted item.

**Usage:** `abcd capture promote <iss-N> [--grounds "<token>: <text>"] | promote <rdi-N> [flags]`

**Flags:**

```
      --grounds string           optional; recorded when given — the conjecture being acted on, not the route taken: "<pursued|deferred|declined>: <what is expected, and what would show it wrong>"
      --intent string            link mode: link this existing itd-N instead of minting a draft (writes both halves of the join)
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
```

**Example:**

```
abcd capture promote iss-2609010000000001
```

#### `abcd capture reframe`

Record a reframe a reading occasioned: Writes one rfm-N fingerprinting the frame before and after; refuses an uncommitted occasion or frame edit without --open.

**Usage:** `abcd capture reframe --occasioned-by <rdi-N|dsp-N|srp-N> --grounds "<why>" [--open] | --complete <rfm-N> [flags]`

**Flags:**

```
      --complete string        the open reframe record (rfm-N) to finish once the rewrite is committed
      --grounds string         why the frame moved (free text, held to the grounds floor)
      --occasioned-by string   the record that occasioned the reframe: a reading item (rdi-N), a disposition (dsp-N) or a surprise (srp-N)
      --open                   record the first half before the rewrite is committed; complete it after with --complete
```

**Example:**

```
abcd capture reframe --occasioned-by rdi-2609010000000001 --grounds "the reading showed the construal assumed a single operator" --open
```

#### `abcd capture remedy`

Write or replace the fix an open issue proposes: Writes the issue's remedy; refuses an empty text, the automatic filers' value or an issue that is not open.

**Usage:** `abcd capture remedy <iss-N> <text>`

**Example:**

```
abcd capture remedy iss-2609010000000001 "make the map before the first write"
```

#### `abcd capture resolve`

Move an open issue to resolved/, naming what fixed it: Writes the moved record; refuses without --impact or on an id this ledger does not hold.

**Usage:** `abcd capture resolve <iss-N> <note> --impact <additive|breaking|fix|internal> [--grounds "<token>: <text>"] [--intent itd-N] [--spec spc-N] [--commit sha] [--shipped-in vX.Y.Z] [flags]`

**Flags:**

```
      --commit string            resolved_by provenance: the fixing commit sha (7-64 hex chars, shape-checked only)
      --grounds string           optional; recorded when given — the conjecture being acted on, not the route taken: "<pursued|deferred|declined>: <what is expected, and what would show it wrong>"
      --impact string            product impact: additive|breaking|fix|internal (required)
      --intent string            resolved_by provenance: the itd-N that fixed it (must exist)
      --production-mode string   restamp how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: leave the record's existing stamp alone; refused on a record that predates disclosure)
      --shipped-in string        MIGRATION USE: the release that already carried this work (vX.Y.Z), leaving the record out of the current cut; unnecessary in a repo abcd managed from the start
      --spec string              resolved_by provenance: the spc-N that fixed it (must exist)
```

**Example:**

```
abcd capture resolve iss-2609010000000001 "fixed by the parser change" --impact fix
```

#### `abcd capture surprise`

Record one surprise a reading item, admission or disposition occasioned: Writes one srp-N record; refuses an unresolved occasion or a text below the floor.

**Usage:** `abcd capture surprise --occasioned-by <rdi-N|adm-N|dsp-N> "<what was unexpected>" [flags]`

**Flags:**

```
      --occasioned-by string   the record that occasioned it: a reading item (rdi-N), an admission (adm-N) or a disposition (dsp-N)
```

**Example:**

```
abcd capture surprise --occasioned-by rdi-2609010000000001 "the proposal nobody expected ranked first"
```

#### `abcd capture wontfix`

Move an open issue to wontfix/ with the reason it is not acted on: Writes the moved record; refuses an id this ledger does not hold.

**Usage:** `abcd capture wontfix <iss-N> <reason> [--grounds "declined: <text>"] [--duplicates <iss-N|itd-N,...>] [flags]`

**Flags:**

```
      --duplicates string        comma-separated iss-N or itd-N ids this issue duplicates, written to its typed duplicates link; each must exist, and a link the record already carries is kept
      --grounds string           override the recorded grounds text (the token stays declined — a wontfix IS that non-action)
      --production-mode string   restamp how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: leave the record's existing stamp alone; refused on a record that predates disclosure)
```

**Example:**

```
abcd capture wontfix iss-2609010000000001 "the behaviour is the documented one"
```

### `abcd changelog`

Preview the next release cut's version, records, and guardrail verdict: Writes nothing; refuses outside a checkout, exiting 0 on a cut the gates would stop.

**Usage:** `abcd changelog`

### `abcd completion`

Generate the autocompletion script for the specified shell

**Usage:** `abcd completion`

Generate the autocompletion script for abcd for the specified shell.
See each sub-command's help for details on how to use the generated script.

#### `abcd completion bash`

Generate the autocompletion script for bash

**Usage:** `abcd completion bash`

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(abcd completion bash)

To load completions for every new session, execute once:

#### Linux:

	abcd completion bash > /etc/bash_completion.d/abcd

#### macOS:

	abcd completion bash > $(brew --prefix)/etc/bash_completion.d/abcd

You will need to start a new shell for this setup to take effect.

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

#### `abcd completion fish`

Generate the autocompletion script for fish

**Usage:** `abcd completion fish [flags]`

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	abcd completion fish | source

To load completions for every new session, execute once:

	abcd completion fish > ~/.config/fish/completions/abcd.fish

You will need to start a new shell for this setup to take effect.

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

#### `abcd completion powershell`

Generate the autocompletion script for powershell

**Usage:** `abcd completion powershell [flags]`

Generate the autocompletion script for powershell.

To load completions in your current shell session:

	abcd completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

#### `abcd completion zsh`

Generate the autocompletion script for zsh

**Usage:** `abcd completion zsh [flags]`

Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(abcd completion zsh)

To load completions for every new session, execute once:

#### Linux:

	abcd completion zsh > "${fpath[1]}/_abcd"

#### macOS:

	abcd completion zsh > $(brew --prefix)/share/zsh/site-functions/_abcd

You will need to start a new shell for this setup to take effect.

**Flags:**

```
      --no-descriptions   disable completion descriptions
```

### `abcd decide`

Mint an ADR id and lay the record's empty skeleton: Writes one proposed record into the decisions store; refuses a missing or unusable title.

**Usage:** `abcd decide "<title>"`

Mint an architecture decision record: allocate its id through the shared record-id
seam and write the store's skeleton under .abcd/development/decisions/adrs/.

The id is `adr-<yymmddHHMMSS><rrrr>` and the filename is ordered by that stamp, so two
branches deciding on the same day cannot allocate the same number — the collision a
hand-numbered ordinal has by construction. The hand-numbered records 0001-0058 keep
their ids and their filenames; nothing is renumbered, and every reader admits both.

The verb writes an EMPTY record: it owns the id, the date, the filename and the four
sections, and states nothing. The decision is the author's to write, and the status it
lands with is `proposed` until the author sets `accepted`.

**Example:**

```
abcd decide "Record ids are minted from a timestamp"
```

### `abcd disembark`

Pack a repository into a lifeboat, probing and planning first: Writes nothing in the source, only inside the lifeboat; refuses an unknown sub-verb.

**Usage:** `abcd disembark`

#### `abcd disembark coverage`

Aggregate saved probe reports into the section-by-repository coverage table: Writes nothing; refuses a file that is not a probe report.

**Usage:** `abcd disembark coverage <report.json>...`

**Example:**

```
abcd disembark coverage probe-report.json
```

#### `abcd disembark graveyard`

Validate host-produced lesson JSON against a packed lifeboat: Writes the lessons that cite their evidence; refuses without --lessons-json.

**Usage:** `abcd disembark graveyard <lifeboat-dir> --lessons-json <file|-> [flags]`

**Flags:**

```
      --lessons-json string   path to the host-produced lesson JSON (or - for stdin)
      --route stringArray     route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

**Example:**

```
abcd disembark graveyard ../lifeboat --lessons-json lessons.json
```

#### `abcd disembark pack`

Pack a lifeboat from a repository into a destination directory: Writes the destination only; refuses when the secret scanner is unavailable.

**Usage:** `abcd disembark pack <repo> <dest> [flags]`

**Flags:**

```
      --include-ignored   also read files git ignores (widens the scan; the report says so)
```

**Example:**

```
abcd disembark pack . ../lifeboat
```

#### `abcd disembark plan`

Show the file set a pack would write: Writes nothing; refuses a repository path that is not a directory.

**Usage:** `abcd disembark plan [repo] [flags]`

**Flags:**

```
      --include-ignored   also read files git ignores (widens the scan; the report says so)
```

#### `abcd disembark press-release`

Compose a lifeboat's press release, or validate the host's: Writes the press-release files in the lifeboat; refuses a host draft citing nothing resolvable.

**Usage:** `abcd disembark press-release <lifeboat-dir> [--press-release-json <file|->] [flags]`

**Flags:**

```
      --press-release-json string   path to host-produced press-release JSON (or - for stdin); absent runs deterministic mode
      --route stringArray           route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

**Example:**

```
abcd disembark press-release ../lifeboat
```

#### `abcd disembark principles`

Distil a lifeboat's principles from its ADRs, or validate the host's: Writes the principles files in the lifeboat; refuses a directory that is not a lifeboat.

**Usage:** `abcd disembark principles <lifeboat-dir> [--principles-json <file|->] [flags]`

**Flags:**

```
      --principles-json string   path to host-produced principle JSON (or - for stdin); absent runs deterministic mode
      --route stringArray        route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

**Example:**

```
abcd disembark principles ../lifeboat
```

#### `abcd disembark probe`

Report which brief sections a lifeboat could ground from a repository: Writes nothing; refuses a repository path that is not a directory.

**Usage:** `abcd disembark probe [repo] [flags]`

**Flags:**

```
      --include-ignored   also read files git ignores (widens the scan; the report says so)
```

#### `abcd disembark review`

Review a packed lifeboat against its source repository, or validate the host's verdict: Writes the review in the lifeboat; refuses an unregistered verdict.

**Usage:** `abcd disembark review <lifeboat-dir> <source-repo> [--review-json <file|->] [flags]`

**Flags:**

```
      --review-json string   path to the host-produced review verdict JSON (or - for stdin); absent runs deterministic mode
      --route stringArray    route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

**Example:**

```
abcd disembark review ../lifeboat .
```

### `abcd docs`

Keep the citation baseline that `abcd lint docs` enforces offline: Writes nothing but that baseline; refuses an unknown sub-verb.

**Usage:** `abcd docs`

#### `abcd docs cite`

Keep the citation baseline the docs lint enforces offline: Writes nothing bare, and only that baseline; refuses an unknown sub-verb.

**Usage:** `abcd docs cite`

##### `abcd docs cite confirm`

Record that a person verified a cited URL the fetcher could not read: Writes a dated manual entry in the baseline; refuses a URL the docs do not cite.

**Usage:** `abcd docs cite confirm [url...] [flags]`

Record that a human verified a cited URL the fetcher could not read.

Name the URLs directly, or pass --receipt with a receipt file. Both write the same dated manual entry: the baseline records THAT a human confirmed the citation and WHEN, never how. Only URLs the documentation actually cites can be confirmed.

**Flags:**

```
      --config string    path to docs-lint.json (default: <root>/.abcd/docs-lint.json)
      --receipt string   path to a JSON receipt listing the confirmed citations: schema_version 1 and a confirmed list, each entry a url with an optional final_url and verified_on (YYYY-MM-DD)
      --root string      repo root (default: current working directory)
```

##### `abcd docs cite refresh`

Fetch every cited URL once, the one documentation verb that reaches the network: Writes the citation baseline; refuses an unreadable docs-lint configuration.

**Usage:** `abcd docs cite refresh [flags]`

Fetch every cited URL once and rewrite the committed citation baseline.

This is the only abcd verb that reaches the network on behalf of documentation. Each URL gets exactly one bounded attempt; a failure is recorded as an outcome, never retried. Sources that refuse automated fetchers are printed as a manual checklist for `abcd docs cite confirm` rather than recorded as broken.

**Flags:**

```
      --config string   path to docs-lint.json (default: <root>/.abcd/docs-lint.json)
      --root string     repo root (default: current working directory)
```

### `abcd drain`

Sort the open issues by the drain's field rule, eligible first in drain order: Writes nothing; refuses to start without --dry-run, as the run is not built.

**Usage:** `abcd drain [flags]`

Work the open issue ledger unattended: fix the issues that need no decision, and
hand the rest back by kind. The rule for which issues need no decision is a
recorded decision, and it reads the record's fields alone: nothing open in
blocked_by; a category in the fixable set (tech-debt, documentation,
inconsistency, drift, bug, ux); severity nitpick or minor; and a remedy: field.
A security issue is always a person's. Every other open issue is handed back,
listed as ineligible, or skipped naming its blocker, by the rule that excluded it.

--dry-run shows every open issue's disposition, the eligible ones first in the
order a drain takes them (category tech-debt, documentation, inconsistency,
drift, bug, ux; then nitpick before minor; then oldest first), and writes
nothing. The host judgement over each eligible remedy does not run in a dry
run; it can only ever hand an issue back.

The run itself is not built: without --dry-run the verb refuses to start, and
exits 2 with nothing read or written.

**Flags:**

```
      --dry-run   show every open issue's disposition and the order a drain takes them; writes nothing
```

**Example:**

```
abcd drain --dry-run
  abcd drain --dry-run --json
```

### `abcd embark`

Unpack a verified lifeboat into a target repository, probing first: Writes only its record families and marker block; refuses the whole write on any conflict.

**Usage:** `abcd embark`

#### `abcd embark from`

Unpack a lifeboat's record families into a target repository: Writes those families and the marker block; refuses the whole write on any conflict.

**Usage:** `abcd embark from <lifeboat-dir> [target-dir]`

**Example:**

```
abcd embark from ../lifeboat
```

#### `abcd embark probe`

Report what a lifeboat would write into a target, coverage blanks first: Writes nothing; refuses a lifeboat whose manifest does not verify.

**Usage:** `abcd embark probe <lifeboat-dir> [target-dir]`

**Example:**

```
abcd embark probe ../lifeboat
```

### `abcd guard`

Judge a shell command against the hazard registry before it runs: Writes nothing; refuses a hazard through check or hook, and an unknown sub-verb.

**Usage:** `abcd guard`

#### `abcd guard check`

Judge one shell command against the hazard registry: Writes nothing; refuses a hazard with exit 1 and a command it cannot parse with exit 2.

**Usage:** `abcd guard check [flags]`

Evaluates one candidate command line against the hazard registry — the
bundled defaults merged with this repo's `.abcd/guard.json` — and reports
allow, warn, or block. A blocker exits 1 and names the safe successor; a
warn exits 0 with the warning rendered; an allow exits 0. A guard that
cannot be evaluated at all (an unparsable command line, a malformed
registry) exits 2, so a caller never reads silence as clearance. Unparsable
means an unterminated quote in COMMAND text, which no shell runs either;
an unterminated quote inside a here-document body is document text and is
not one. Grammar a shell does run gets a verdict instead: a trailing
backslash is read as bash reads it, and a here-document whose delimiter
line never comes is a block, because the rest of the input may be commands
the guard did not check.

Matching is shell-token-aware and applies in command position only, so a
hazard named inside a quoted argument never fires.

The guard is a MISTAKE FILTER, not a security boundary. It catches a hazard
typed by accident or reached through an ordinary wrapper — the cases that
actually cost people work. It does not withstand an author trying to get a
command past it, and it does not claim to: the set of programs that launch
another program is open-ended, so no list inside this binary can enumerate
it, and a repository extends that set with one line in a Makefile. Anything
that needs an enforced boundary needs a control at the execution layer — a
sandbox, a permission system, a restricted shell — with this guard in front
of it to teach, never in place of it.

An allow means no registry entry matched — it is never a statement that a
command is safe. A hazard behind a launcher the guard does not recognise is
a WARN naming the entry it matched, rather than an allow, because the guard
cannot tell whether that program runs the rest of the line. A `$(…)`,
backtick, `<(…)` or `>(…)`, quoted or not, IS followed into command
position, and the words written after one stay the enclosing command's,
so `rm $(true) -rf *` is read as `rm -rf *`. What one prints is unknown,
so a word holding one fails closed, read every way it can be at once: led
by a dash (`--$(…)`) it is every flag it could become — standing alone,
taking a value, a shell's `-c` — before the command as well as after it;
after a value flag (`git -C $(pwd) push`) it is that flag's value; as an
operand it is one operand; in command position (`$(echo git) push`) it is
any program its known text allows, so an unknown name with any operand
reads as `pkill` too; a block that fires only on such a name is reported
as program-name-unknown, and the way past is to spell the program's name.
Text beside one in the same word is also read as bash
leaves it when the output is empty. One nested more than eight
double-quoted substitutions deep, holding a case command, or more than
eight of them where the program name could be, is blocked, because the
guard has stopped reading it. An ANSI-C string ends at its closing quote
and its first NUL, as bash ends it. A `${…}` holding a substitution is
unknown from its `${` on, and inside double quotes it ends at its own
`}`, its nested quotes opening a nested string. A here-document body is
data, but a substitution in one whose delimiter is unquoted runs, and is
read as a command; a body line ending in an odd number of backslashes
joins the next before the delimiter compare, as bash joins it. A
backtick's text is read after bash's own pass over it, which drops a
backslash before `$`, a backtick or a backslash (and, directly inside
double quotes, one before a `"` too), so an escaped `\$(…)`
or an escaped backtick pair between backticks is read as the
substitution bash runs, in a here-document body there too. A
`"$(cat <<'EOF' … EOF)"` handed to `sh -c` or `eval` is read as its
document's text, and an unquoted one as the words bash splits its
document into, at every layer, each joined to any text written
beside it in the same word, as bash joins it; a backtick spelling with
no backslash in it is read the same way. On a line where another command
names IFS an unquoted one is blocked (ifs-split-unread), because the
guard splits on the default IFS only. Two `sh -c` or `eval` layers are
followed; a payload nested deeper is blocked.
`$(( … ))` is an expression, not commands. A shell reading
its script from a pipe, a here-document, a here-string, the stdin device
or a process substitution is blocked, and so is a line over 64 KiB.
An unquoted brace group IS
expanded as bash expands it, and one past 4096 words is blocked. What an
allow still does not see is a hazard that never reaches command position at
all: a word that is wholly a `$(…)` standing where a flag would be (read as
an operand, the way a commit message or a branch is spelled), a delete
target printed whole by one (`rm -rf $(echo /)`, read by its known text
the way `rm -rf $(find …)` names its targets every day), one launched
through a known
wrapper carrying a value-taking flag the guard does not name (`sudo -u bob
<hazard>` is seen; the bundled short form `sudo -Hu bob <hazard>` reaches
only the warn, not the entry that names it),
one whose API path an entry names by its ROOT segment but the host serves
under a prefix (a GitHub Enterprise Server install mounts the same endpoints
under `/api/v3/`; the api.github.com URL form IS read), a parameter
expansion that carries no substitution (`$VAR`, `${VAR:-git}`) wherever it
stands — as the program's name, as a flag (`--$VAR`), or inside an
interpreter payload (an execute-a-string payload IS read — `sh -c`,
`env -S`; one the guard cannot read is warned or, for `env -S`, blocked),
because the guard sees the variable, not what the shell expands it to,
an IFS the shell already holds when the line starts or gains during the
line through a name the guard does not read (every line is read from the
default IFS),
a hazard inside a NON-shell interpreter's payload (`python -c`, `perl -e`) —
one opaque token the tokenizer cannot read, today a silent allow (a warn for
it is a recorded design target, not yet raised),
a lone substitution standing as the whole command (`$(cat msg.txt)`,
`$(date)`), which can be any program but matches no entry with no
operand after it, and so a document printed that way through any shape
but exactly `cat <<DELIM`, a newline, the body, the delimiter line and
blanks (`/bin/cat`, `command cat`, `cat -`, a redirection or a command
beside it, a backslash-newline in it, a `${…}` around it),
or a dangerous form no entry describes. Coverage is what the registry
names.

The candidate comes from --command, or from stdin when the flag is absent.
Prefer stdin for a command line you did not type yourself: the shell expands
a double-quoted --command argument before this verb starts, so a candidate
containing a command substitution would run at check time. A quoted-delimiter
heredoc (`abcd guard check <<'EOF'` ... `EOF`) passes it through untouched.

**Flags:**

```
      --command string   the candidate command line (default: read from stdin)
```

#### `abcd guard hook`

Judge the shell command in a host's pre-tool-use payload: Writes nothing; refuses a hazard with the host's blocking status.

**Usage:** `abcd guard hook`

Reads a host pre-tool-use hook payload on stdin and evaluates its shell
command against the hazard registry. A blocker exits with the host's
blocking status and puts the safe successor and the plain-language why on
stderr, which is the channel the host replays to the agent. A warn and an
allow both let the command run.

Anything the adapter cannot turn into a decision — an unreadable payload, a
tool call that is not a shell command, a registry that will not load —
allows the command and warns loudly on stderr. A guard that cannot answer
never stops a session, and is never silently absent. A command line the
guard cannot split is not in that set: it is blocked (command-unparsable),
because a line the guard misreads may be one bash runs, and letting it
through would pass every hazard in it. Unparsable means an unterminated
quote in COMMAND text, which no shell runs either — a quote inside a
here-document body is document text and is not one. A trailing backslash
and a here-document with no delimiter line are grammar a shell does run,
so each gets a verdict — the backslash is read as bash reads it, the
unterminated document blocks.

The hook judges only what the host hands it, and the plugin's hook
manifest hands it the shell tool and the question tool and nothing else.
A call through any other tool never reaches the guard: a file the host's
own tools write or edit, or a command a tool from another extension runs,
is neither checked nor warned about. That is the guard's standing scope,
not a degradation, and the guard: line of abcd ahoy does not report it.

A host whose shell tool takes a per-call working directory passes it as
tool_input.workdir. It is resolved against the session directory, and a
command whose workdir is an existing directory in another repository is
checked against that repository's registry as well as the session's; the
stricter verdict wins, so the workdir's registry can add a hazard and never
remove one. The workdir is never read as a cd: the one host that has the
field fails the call when the directory is missing, so no failed-cd hazard
exists. A workdir that is not a string, or holds a NUL byte, a control
character or invalid UTF-8, or is over 4096 bytes, is refused with the
blocking status and the reason.

On the host's question tool the hook gates the question on the mode, not
the registry. In a checkout abcd manages, a question asked while `abcd mode`
reads managed is refused with the blocking status, naming `abcd mode
product-thinker` and `abcd mode facilitator`; once the mode names somebody
the question runs and is marked open in the local tier, and the next human
message resets the mode to managed. Elsewhere a question runs unchecked.

### `abcd help`

Help about any command

**Usage:** `abcd help [command]`

Help provides help for any command in the application.
Simply type abcd help [path to command] for full details.

### `abcd history`

Keep session transcripts in the user-level store and read them back: Writes nothing bare, and redacts each one it stores; refuses an unknown sub-verb.

**Usage:** `abcd history`

#### `abcd history capture`

Redact and store a session transcript, or a whole session with --all: Writes one record per transcript; refuses stdin or --all without --session.

**Usage:** `abcd history capture [<transcript-file> | - | --session <id> --all <path>...] [flags]`

**Flags:**

```
      --all              capture every transcript of the --session named — its main thread and each sub-agent — found under the paths given (default: ingest_roots)
      --kind string      source route: native (abcd's own capture of the host's transcript) | import (another tool's export) (default native)
      --session string   session id for the record (default: transcript filename; required for stdin)
      --tool string      source tool: the tool that produced the transcript, a lowercase slug (default host on a native capture; required on an import)
```

#### `abcd history discard`

Delete one staged or quarantined raw transcript for good: Writes the deletion; refuses without --yes.

**Usage:** `abcd history discard <staged-filename> --yes [flags]`

**Flags:**

```
      --yes   confirm the irreversible deletion of an unredacted transcript
```

**Example:**

```
abcd history discard 0123abcd-session.raw --yes
```

#### `abcd history drain`

Redact and store every transcript staged for this repository: Writes the records into the store; refuses outside a git checkout.

**Usage:** `abcd history drain`

#### `abcd history ingest`

Redact and store transcripts already on disk into a named repository: Writes that repository's store; refuses without --into.

**Usage:** `abcd history ingest [<path>...] [flags]`

**Flags:**

```
      --adopt stringArray   project directory name to claim for this run, in addition to adopt_projects (repeatable)
      --into string         destination repository root (REQUIRED, no default; its own redaction configuration governs everything stored)
```

#### `abcd history list`

List this repository's stored transcripts, newest first: Writes only a missing store and a legacy corpus moved into it; refuses outside a git checkout.

**Usage:** `abcd history list [flags]`

**Flags:**

```
      --session string   list one session's whole set — its main-thread record and every sub-agent it spawned, main thread first
```

#### `abcd history migrate`

Repair records filed under a composite session id: Writes a missing store, and the repaired records only with --apply; refuses outside a git checkout.

**Usage:** `abcd history migrate [flags]`

**Flags:**

```
      --apply                      write the repaired records (default: report only)
      --sidecar-root stringArray   directory to search for the harness's per-agent metadata (repeatable; default: ingest_roots from .abcd/config/history.json)
```

#### `abcd history reconstruct`

Render one session and its sub-agents as one artefact plus telemetry: Writes both files into --out; refuses an --out that is not an existing directory.

**Usage:** `abcd history reconstruct <session-id> [flags]`

**Flags:**

```
      --max-block-bytes int   truncate one rendered tool input or result at this many bytes (0 disables); what is removed is marked and counted (default 8192)
      --mode string           full (every turn of every agent) | spine (the main thread whole, each sub-agent reduced to its instruction and its conclusion) (default "full")
      --out string            directory to write <session>.md and <session>.telemetry.json into, or - for stdout (default ".")
```

**Example:**

```
abcd history reconstruct 0123abcd-session
```

#### `abcd history separation`

Report whether any retained transcript held both a reading and the ledger of one run: Writes nothing; never refuses, exiting 1 naming each such transcript.

**Usage:** `abcd history separation`

#### `abcd history show`

Show one stored transcript's metadata and redacted body: Writes only a missing store and a legacy corpus moved into it; refuses an id the store does not hold.

**Usage:** `abcd history show <session-id-or-filename>`

**Example:**

```
abcd history show 0123abcd-session
```

#### `abcd history staged`

List the ended transcripts not yet redacted into the store: Writes only a missing store and a legacy corpus moved into it; refuses outside a git checkout.

**Usage:** `abcd history staged [flags]`

**Flags:**

```
      --all-repos   survey every repository in the store, not just this one
```

### `abcd ideate`

Judge an idea through the host-run admission gauntlet: Writes nothing bare, and one research record and its decision-log line; refuses an unknown sub-verb.

**Usage:** `abcd ideate`

Record the verdict of abcd's idea-admission protocol — primary-source research, a grill
against the existing record, and an independent adversarial review.

The legs are host work; `/abcd:ideate` orchestrates them. This verb validates what they
produced and writes the durable verdict. Ideate is OPTIONAL and never a gate: no other
verb requires it, and skipping it is never warned about.

#### `abcd ideate record`

Validate a host-composed gauntlet verdict: Writes the dated research record; refuses without an idea slug or --verdict-json.

**Usage:** `abcd ideate record <idea-slug> --verdict-json <file|-> [flags]`

**Flags:**

```
      --verdict-json string   path to the host-composed verdict JSON (or - for stdin)
```

**Example:**

```
abcd ideate record widen-the-public-api --verdict-json verdict.json
```

### `abcd identity`

Record the identity block and propose drift corrections: Writes nothing bare, only the block and its pointer; refuses bare, naming `abcd lint identity`.

**Usage:** `abcd identity [command]` (the bare form's work is `abcd lint identity`)

#### `abcd identity init`

Record this repository's identity block and the pointer to it: Writes the block and the pointer; refuses without --title and --tagline when no block exists.

**Usage:** `abcd identity init [flags]`

**Flags:**

```
      --file string      repo-relative file the identity block lives in (default .abcd/development/IDENTITY.md)
      --heading string   heading the identity block sits under (default "Identity (canonical)")
      --pitch string     the project's short elevator pitch (optional)
      --tagline string   the project's one-line tagline (required unless a block already exists)
      --title string     the project's title (required unless a block already exists)
```

#### `abcd identity render`

Print the correction for every drifted surface as a unified diff: Writes nothing; refuses a repository that records no identity block.

**Usage:** `abcd identity render`

### `abcd implement`

Share one autonomous run between sessions and drive the implement loop: Writes nothing bare, only the run state its sub-verbs name; refuses an unknown sub-verb.

**Usage:** `abcd implement`

The run machinery an autonomous run calls. The shared run lives in the machine-scoped run
state, `~/.abcd/runs/<root-sha>/`, keyed on the repository's root commit, so sessions
in different worktrees of one repository share one run and no repository file.

Bare `abcd implement` is read-only: the sessions that have joined, the claims and
whether each lease still holds, and the window's division mode. It creates nothing.

A session joins (`join`), which writes its record and a session_open line; nothing
signals any other session. The first session opens each window with its mode
(`mode`). A session claims a record before opening its lane (`claim`): one file per
record, taken by an exclusive create, so of two sessions reaching for one record
exactly one holds it; the claim is a lease, and a lapsed lease is claimable again.
The second session is bounded: one lane at a time, never the release, never a lane
that touches the reading corpus, no lane in a split-roles window (`check` asks before
a step that is not a claim). `log` appends the run's other events, and `report`
derives the comparison of the modes from the log.

`status`, `step` and `receipt` drive the implement loop `abcd build` starts, whose state
lives in this checkout's local tier: `step` performs one step and exits, naming the
agent, brief and receipt path when a step hands work to an agent, and `receipt`
completes that step once the receipt verifies.

Exit 2 on a refusal (an unrecognised input, a session that has not joined, a bound
the session's role does not permit), exit 3 on contention (the record is claimed by
another session, or the run state is locked): back off and take other work.

#### `abcd implement check`

Ask whether this session may take a step before taking it: Writes a run-log line only on a refusal; refuses a step the second session's bounds forbid.

**Usage:** `abcd implement check <lane|release|review|audit|land> --session <id> [flags]`

Say whether this session may take a step, before it takes it. The first session may
take every step. The second is refused the release step always, a lane in a
split-roles window, and a lane whose --path reaches the reading corpus; review,
audit and land are open to it. A refusal exits 2 and is logged; an allowed step
writes nothing. The verdict reports the agent ceiling the session joined with and the
agents its log lines declare alive (agents_alive).

**Flags:**

```
      --path stringArray   a repository-relative file the step touches (repeatable)
      --session string     this session's id
```

**Example:**

```
abcd implement check lane --session s-example
```

#### `abcd implement claim`

Claim a record for this session before opening its lane: Writes the claim and a run-log line; refuses a record another session holds.

**Usage:** `abcd implement claim <record> --session <id> --lane <lane> [flags]`

Take a record for this session: one claim file per record in the run state, created
exclusively, so of two sessions reaching for one record exactly one holds it. The
claim is a lease (--lease, default 2h, 1m to 24h). Claiming a record this session
already holds renews the lease. A claim whose lease has passed is claimable again,
and the lapse is logged as claim_lapsed. A record another session holds is refused
at exit 3 and logged as claim_denied naming the holder; the second session also
logs a backoff with its reason and the minutes the attempt spent. A run state
locked by another session's change is exit 3 too, and the second session's
backoff from it is logged the same way.

The second session is refused (exit 2, logged as a refusal) when it already holds
a live claim, when the window is split-roles, or when a --path it declares is in the
reading corpus.

**Flags:**

```
      --lane string        the lane the claim is for
      --lease duration     how long the claim holds before it lapses (1m to 24h) (default 2h0m0s)
      --path stringArray   a repository-relative file the lane will touch (repeatable); checked against the reading corpus
      --session string     this session's id
```

**Example:**

```
abcd implement claim iss-2609010000000001 --session s-example --lane cli
```

#### `abcd implement join`

Join the run with a stated role: Writes the session's record and a session_open line; refuses the other role on a resume.

**Usage:** `abcd implement join --session <id> --role first|second [flags]`

Record this session in the run state with its role and log a session_open line.
Nothing signals any other session: the first learns of a second only by reading the
run state. Joining again with the same role is a resume and is logged as one; asking
for the other role is refused. The role is the session's own statement, recorded
here and read by every bound — never taken from the environment.

--ceiling states the session's own agent ceiling: the most agents it runs at once (for
the second session, on top of the first session's). abcd runs no agent: it counts the
agents the session's own agent_start and agent_end lines declare alive, refuses an
agent_start past the ceiling, and reports the count with every `check`. An agent the
session never logs is invisible to it. A resume keeps the ceiling.

**Flags:**

```
      --ceiling int      this session's own agent ceiling (1 to 64; 0 states none), held against its logged agent_start lines
      --model string     the model this session runs, recorded on the session_open line
      --reason string    why the session opens (run start, window, resume), recorded on the line
      --role string      first | second
      --session string   this session's id (letters, digits, '.', '_', '-')
```

**Example:**

```
abcd implement join --session s-example --role first
```

#### `abcd implement leave`

Leave the run, releasing every claim this session holds: Writes the releases and a session_close line; refuses without --session.

**Usage:** `abcd implement leave --session <id> [flags]`

Release every claim this session holds (each logged as claim_released), log a
session_close line with the reason, and remove the session's record. A session
that stops without leaving strands nothing: its claims lapse with their leases.

**Flags:**

```
      --reason string    why the session closes (window, stop condition, crash recovery)
      --session string   this session's id
```

**Example:**

```
abcd implement leave --session s-example
```

#### `abcd implement load`

Check the machine's load before abcd's own tests start: Writes a load event to the run log inside a run; refuses an unknown --site, never a loaded machine.

**Usage:** `abcd implement load --site preflight|eval-harness [flags]`

Read the machine's load averages and process table once and warn when a program
outside the running work has used nearly all the CPU it could get for longer than
the stray limit, or when the one-minute load average is above the extreme limit.
What a program could get is its fair share: the online cores divided by the
one-minute load, and never more than one core. A lifetime CPU share of at least 0.9
of it makes a stray, so forty busy loops on 16 cores, each at 0.4 of a core, are all
strays, as one loop at a full core of an idle machine is. Your programs and other
accounts' are judged alike. `make preflight` runs it first, and the eval harness runs
it once at its start; it never runs once per test package. It never refuses, never
waits and never stops anything, and it exits 0 on every status: ok, warning, skipped
(in CI, where the line says why) and unchecked (a platform other than macOS and
Linux, or a read that failed).

Your own strays are named with their pid, process group, age and CPU share, with
commands to stop them that re-check each target first and never match by pattern;
names the private banned-names layer matches are masked. Other accounts' strays
appear only as a count and a total CPU share. The check's own parent chain is never
a stray. Inside an autonomous run (a run state with a joined session) a warning is
also written to the run log as a `load` event.

The limits are per machine, in `~/.abcd/load-limits`, which the check reads and
never creates. `#` starts a comment; every other line is `<key> <value>`:

  stray-minutes 30   minutes at nearly all its share before a program is a stray (1 to 10080)
  extreme-load 64    the one-minute load above which the machine is overloaded

Either key may be omitted. The defaults are 30 minutes and four times the online
core count. The file must be a regular file you own that nobody else can write, at
most 4 KiB; a file that is not, or that holds an unknown key, a repeated key or a
value out of range, is reported loudly and both defaults are used.

**Flags:**

```
      --site string   where the check runs: preflight | eval-harness
```

**Example:**

```
abcd implement load --site preflight
```

#### `abcd implement log`

Append one of the run's events to today's run log: Writes one line; refuses the claim, window, and session events their own verbs write.

**Usage:** `abcd implement log <event> --session <id> [--field key=value ...] [flags]`

Append one event line to today's run log (`~/.abcd/runs/<root-sha>/<UTC date>.jsonl`)
in a single append, so two sessions writing at once each land whole lines. The line
carries ts, session and event, then each --field. A value that reads as a number or
a boolean is written as one when it reads back as the same text, so `sha=0123456`
stays a string. The events: backoff, lane_open, lane_close, agent_start, agent_end, ceiling_wait, gate_run, review, fallback, stop, refusal, pr, capture, context, ceiling_overrun, intervention, decision.
The claim, window and session events are written by their own sub-verbs and are
refused here, so the log cannot record a claim the run state does not hold.

An event missing a field the report reads is refused, naming it: lane_close (lane, outcome); agent_start (agent); agent_end (agent, role, model, minutes|wall_minutes|wall_min); stop (cause); ceiling_overrun (alive, ceiling, lane, minutes); intervention (kind, by, what, why, autonomy_gap); decision (what, alternative, why).
A backoff names its reason and the minutes it spent (reason=<why>, minutes=<n>),
or it is refused. An intervention's kind is one of session_open, account, ruling, restart, close_session, file_restore, permission, other; an at or
last_productive is an RFC 3339 time, and a *_min or minutes field a number. An agent_start
that would take a session past the ceiling it joined with is refused, and the refusal logged.

**Flags:**

```
      --field stringArray   an event field as key=value (repeatable)
      --session string      this session's id
```

**Example:**

```
abcd implement log lane_open --session s-example
```

#### `abcd implement mode`

Open a window by logging its division mode: Writes a window_mode line; refuses any session but the first.

**Usage:** `abcd implement mode <single|claim|batch|split-roles> --session <id> [flags]`

Log a window_mode line naming how this window divides the work: `single` (one
session), `claim` (a session claims a record before opening its lane), `batch` (the
run file assigns whole batches per session), or `split-roles` (the first session
builds; the second reviews, audits and lands). Only the first session sets it. The
mode in force is the log's last window_mode line, whoever wrote it.

**Flags:**

```
      --session string   this session's id (a first session)
      --window int       the window's number, recorded on the line
```

**Example:**

```
abcd implement mode single --session s-example
```

#### `abcd implement receipt`

Hand back the receipt an agent step of a loop run awaits: Writes the run's state when the receipt verifies; refuses a receipt that does not verify.

**Usage:** `abcd implement receipt <path> [--run <run-id>] [flags]`

Hand back the receipt the run's awaiting lane named when its step handed work to an
agent. The path must be the one the step named. The step's verifier checks it; a
receipt that verifies completes the step and the lane moves to its next step, and one
that does not is refused naming what is missing, with the lane left where it was. A
step whose verifier this abcd does not carry is refused naming the spec piece that
delivers it.

An implementer's receipt is read strictly (one JSON object, no field the brief does not
name, within its size cap, never through a symlink) and verifies only when every commit
it names is on the lane's branch past its base, the definition of done's output exists
in the lane's directory with a zero exit code, and the report exists there. A receipt
that verifies moves the lane's head to its branch's tip.

--run names the run; without it, the one run in progress in this checkout. Exit 2 on a
refusal, exit 3 on a locked run state.

**Flags:**

```
      --run string   the run the receipt belongs to (run-<16 digits>); the one run in progress when omitted
```

**Example:**

```
abcd implement receipt review-receipt.json --run run-2609010000000001
```

#### `abcd implement release`

Release this session's claim on a record: Writes the release and a claim_released line; refuses a claim another session holds.

**Usage:** `abcd implement release <record> --session <id> [flags]`

Remove this session's claim on a record and log claim_released. Only the holder
releases a claim; another session's claim lapses with its lease instead.

**Flags:**

```
      --session string   this session's id
```

**Example:**

```
abcd implement release iss-2609010000000001 --session s-example
```

#### `abcd implement report`

Derive the comparison of the division modes from the run log: Writes nothing; refuses --date and --log together.

**Usage:** `abcd implement report [--date YYYY-MM-DD | --log <file>] [flags]`

Derive, per division mode, the figures the run's report compares: windows, wall
clock, lanes opened and landed (a lane_close whose outcome is merged or landed),
the second session's lanes landed, collisions (claim_denied), lapsed claims,
backoffs and the minutes backed off, agent minutes (agent_end's minutes, wall_minutes
or wall_min), ceiling wait, ceiling overruns and refusals, per session within each mode.
Each event belongs to the window open when it happened, and a join logged at most a
minute before a window_mode to that window; each session's context lines are totalled
across the run, with the last used_pct seen. `leader` is the mode with the most lanes landed per wall-clock hour —
a figure, not a verdict. Over the whole run it counts the evidence (interventions by
kind, stops, decisions), names the lines lacking a field `log` requires of their event
(missing_fields), and names each of lane_open, lane_close, agent_start, agent_end and
gate_run whose lines stop more than six hours before the run's last line (coverage).
Lines the reader cannot use are listed, never dropped silently.

By default the run's whole log is read, every day of it; --date reads one day, and
--log reads one log file named directly. Reads only; creates nothing.

**Flags:**

```
      --date string   read one day's log (YYYY-MM-DD, UTC)
      --log string    read this log file instead of the run's own
```

#### `abcd implement status`

Render the implement loop's runs in this checkout, lane by lane: Writes nothing; refuses a --run naming no run.

**Usage:** `abcd implement status [--run <run-id>] [flags]`

Render the runs `abcd build` started in this checkout, or the one --run names: the
intent and spec, each lane with its spec step and next step, what an awaiting lane
waits on, the pending spec steps, and the run record. Read-only: it writes nothing
and creates nothing. Exit 2 when --run names no run.

**Flags:**

```
      --run string   the run to render (run-<16 digits>); every run in this checkout when omitted
```

#### `abcd implement step`

Perform the next step of an implement loop run and exit: Writes the run's state, the lane's worktree or brief; refuses a step this abcd does not carry.

**Usage:** `abcd implement step [--run <run-id>] [flags]`

Perform one step of the run's current lane, write the state, and exit. At a step that
hands work to an agent, the result names the agent to start, the brief it is handed
and the path its receipt goes to; the lane then advances only on
`abcd implement receipt`, and asking for a step again re-tells the same thing and
moves nothing. When a lane is done the spec's next pending step opens the next lane,
and the run record names it. A complete run says so.

The lane's steps, in order: worktree makes the lane's worktree in the machine-scoped
store, ~/.abcd/worktrees/<root-sha>/<run-id>-<lane-id>, on a branch build/<run-id>-<lane-id>
cut from the default branch; brief renders the lane's brief from that base (the intent,
the spec, the conventions of AGENTS.md, the decisions the intent cites, and the spec
steps before the lane's with what landed each) into the lane's directory of the run;
implement hands the lane to a fresh implementer and awaits
its receipt; validate and land follow.

A step whose body this abcd does not carry is refused naming the spec piece that
delivers it, and the run is unchanged. A step that fails leaves the state as it was,
so the next invocation performs it again; a completed step is never repeated.

The run's window clock: once the run's working window has elapsed, the step starts
nothing, writes next_eligible_at (now plus the run's pause) and exits 0 naming it; an
agent already started may still hand back its receipt. Before next_eligible_at the step
is refused as a pause and nothing changes; at or after it, a new window opens.

--run names the run; without it, the one run in progress in this checkout. Exit 2 on a
refusal, exit 3 on a pause or a locked run state.

**Flags:**

```
      --run string   the run to step (run-<16 digits>); the one run in progress when omitted
```

### `abcd inbox`

List the reports managed repositories filed back to abcd, newest first: Writes nothing; refuses any argument.

**Usage:** `abcd inbox`

Read the reports repositories abcd manages filed with `abcd report`, from the
inbox in the user account's machine store (`~/.abcd/inbox/`).

Bare `abcd inbox` lists the waiting reports newest first, naming each sender
repository plainly; `abcd inbox show <id>` renders one whole. Both are
read-only and file nothing. A report written to a template version this abcd
does not know is listed as unreadable, naming the version, and is never
dropped. Everything a report says is another repository's words and is
sanitised before it reaches the terminal.

`abcd inbox promote <id>` is the one act that files anything: it files the
report as a capture in the ledger of abcd's own checkout, through the capture
verb's own path and redactor, with source `managed-repo`. Every report is about
abcd, so run anywhere else it is refused. The capture carries the sender's
root-commit key and the words "a managed repository", never the sender's
name, and the report's id as its evidence; a record id the report names is the
sender's, and is written as one word so it cites nothing in abcd's record. The
report is kept, marked promoted.

Exit 2 on a refusal, with nothing written.

#### `abcd inbox promote`

File one report as a capture in abcd's own ledger: Writes the capture and marks the report promoted; refuses outside abcd's own checkout.

**Usage:** `abcd inbox promote <id>`

**Example:**

```
abcd inbox promote rpt-2609010000000001
```

#### `abcd inbox show`

Render one report whole: Writes nothing; refuses an id the inbox does not hold.

**Usage:** `abcd inbox show <id>`

**Example:**

```
abcd inbox show rpt-2609010000000001
```

### `abcd intent`

File a draft intent from quoted text, or render the intent store's status bare: Writes the draft into drafts/; refuses a lone word.

**Usage:** `abcd intent [text] [flags]`

**Flags:**

```
      --impact string            stamp the draft's product impact: additive|breaking|fix (optional)
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
      --title string             the draft's H1 title (default: the first sentence of the text, cut at the slug cap)
```

#### `abcd intent audit`

List or drain owed fidelity reviews, emit an intent's request, or check issue drift: Writes an OWED stub only for --owed or an id; refuses an unshipped intent.

**Usage:** `abcd intent audit [<itd-N>] | audit --owed [--max <n>] | audit --issue-drift [--strict] [flags]`

**Flags:**

```
      --issue-drift         walk the intent store and the issue ledger for promote joins that do not read the same from both ends (related_issues ↔ related_intents); warns on stderr, exits 0
      --max int             with --owed: list at most n owed reviews (0: no cap); the summary names how many remain
      --owed                drain the owed fidelity reviews: list them oldest shipped first and emit the oldest's request; writes (parks an OWED stub in a markerless intent, a committed record, and rewrites its request); runs no reviewer
      --route stringArray   route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
      --strict              with --issue-drift: exit 1 when any finding is reported (the CI mode)
```

**Example:**

```
abcd intent audit itd-2609010000000001
```

##### `abcd intent audit ingest`

Ingest an intent-audit verdict into the shipped intent: Writes its Audit Notes; refuses without --verdict-json.

**Usage:** `abcd intent audit ingest --verdict-json <path> [flags]`

**Flags:**

```
      --route stringArray     route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
      --verdict-json string   path to the intent-audit verdict JSON, in the shape the Verdict shape section of its review request states
```

**Example:**

```
abcd intent audit ingest --verdict-json verdict.json
```

#### `abcd intent condition`

Read or disposition a shipped intent's scope conditions: Writes a dated condition block; refuses an unresolved occasion or thin grounds.

**Usage:** `abcd intent condition <itd-N> [<cond-id> --disposition <survived|narrowed|falsified|untested> --occasioned-by <rdi-N|itd-N> --grounds "<why>" [--narrowing "<what now holds>"]] [flags]`

**Flags:**

```
      --disposition string     the condition's disposition: survived|narrowed|falsified|untested
      --grounds string         why: held to the grounds substance floor, redacted before it is written
      --narrowing string       what now holds: required on narrowed and refused on every other value
      --occasioned-by string   what occasioned it: a reading item (rdi-N) or a shipped intent (itd-N)
```

**Example:**

```
abcd intent condition itd-2609010000000001
```

#### `abcd intent consistency`

Emit the consistency request over the brief and every intent, or one intent against them: Writes the request locally; refuses a superseded intent.

**Usage:** `abcd intent consistency [<itd-N>] [flags]`

**Flags:**

```
      --route stringArray   route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

##### `abcd intent consistency ingest`

Ingest consistency findings as a dated review and a capture per finding: Writes the report and the ledger records; refuses without --findings-json.

**Usage:** `abcd intent consistency ingest --findings-json <path> [flags]`

**Flags:**

```
      --findings-json string   path to the consistency findings JSON the intent-auditor returned
      --route stringArray      route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

**Example:**

```
abcd intent consistency ingest --findings-json findings.json
```

#### `abcd intent hold`

Hold a draft or planned intent so that planning refuses it: Writes the held line with its reason; refuses without --reason.

**Usage:** `abcd intent hold <itd-N> --reason "<text>" [flags]`

**Flags:**

```
      --reason string   why the record is held: one line, required; redacted before it is written
```

**Example:**

```
abcd intent hold itd-2609010000000001 --reason "waiting on the product thinker's ruling on scope"
```

#### `abcd intent link`

Link a planned intent to an existing spec: Writes the intent's spec_id; refuses an intent that is not planned.

**Usage:** `abcd intent link <itd-N> <spc-N>`

**Example:**

```
abcd intent link itd-2609010000000001 spc-2609010000000002
```

#### `abcd intent plan`

Plan a draft, or several as a named bundle, or stamp a planned one's conditions: Writes the intents and their spec; refuses a held intent or a bundle's blocker.

**Usage:** `abcd intent plan <itd-N> [<itd-N>…] [--bundle <name>] [--target <vX.Y.Z|next>] [flags]`

**Flags:**

```
      --bundle string            the name of the bundle several intents are planned as: kebab-case, required with two or more intents and refused with one
      --impact string            stamp the intent's product impact: additive|breaking|fix (optional; refused when it disagrees with one already recorded)
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
      --target string            the release the planned intent must land by: vX.Y.Z or next, written as target_release (optional; one intent only)
```

**Example:**

```
abcd intent plan itd-2609010000000001
```

#### `abcd intent ready`

Report whether an intent is ready to implement, exiting 1 when not: Writes its grounds only with --grounds; refuses malformed grounds.

**Usage:** `abcd intent ready <itd-N> [--grounds "<pursued|deferred|declined>: <conjecture>"] [flags]`

**Flags:**

```
      --grounds string   record the conjecture behind this gate decision: "<pursued|deferred|declined>: <what is expected, and what would show it wrong>"
```

**Example:**

```
abcd intent ready itd-2609010000000001
```

#### `abcd intent reclassify`

Change an intent's kind, or retire it as superseded by a named successor: Writes the record and its successor together; refuses a shipped intent's kind change.

**Usage:** `abcd intent reclassify <itd-N> --kind <standalone|bundle-member --bundle <name>|superseded --by <itd-M|adr-N> --reason "<why>"> [flags]`

**Flags:**

```
      --bundle string   with --kind bundle-member: the bundle to join, one another record already names
      --by string       with --kind superseded: the successor, an intent (itd-M) or an ADR (adr-N)
      --kind string     the new kind: standalone, bundle-member, or superseded (a discipline is filed, never reclassified into)
      --reason string   why, one line, redacted before it is written; required with --kind superseded
```

**Example:**

```
abcd intent reclassify itd-2609010000000001 --kind superseded --by itd-2609010000000002 --reason "absorbed by the later intent"
```

#### `abcd intent target`

Name the release a planned intent must land by: Writes its target_release line; refuses a draft, a shipped intent, or a value not vX.Y.Z or next.

**Usage:** `abcd intent target <itd-N> <vX.Y.Z|next>`

**Example:**

```
abcd intent target itd-2609010000000001 v0.11.0
```

#### `abcd intent unhold`

Lift an intent's hold: Writes the removal of its held line; refuses a record not held.

**Usage:** `abcd intent unhold <itd-N>`

**Example:**

```
abcd intent unhold itd-2609010000000001
```

### `abcd lab`

List this repository's labs with their pins, probe counts and halts: Writes nothing; refuses outside a git checkout.

**Usage:** `abcd lab`

A lab is a throwaway world pinned at one commit of this repository, run to
answer one question. Its evidence lives at the operator level, in
~/.abcd/lab/<root-sha>/<lab-id>/ (the same root-commit key the transcript
store uses), and its knowledge enters the record only through capture:
nothing any lab verb does writes into the repository.

Bare `abcd lab` lists this repository's labs, read-only: each lab's pin,
probe count, and whether a gate holds it halted.

A gate refusal halts the lab and is recorded as a finding in its findings
log rather than adapted around: the preflight and the retraction sweep exit 1
on a refusal, write their artefact, and hold the lab halted — no probe is
recorded — until the same gate passes again. The finding stays.

Exit 2 on a refusal of the request itself (no checkout, an unknown lab, a bad
name or question); nothing is written on any of them.

#### `abcd lab harvest`

Assemble a lab's harvest with its capture candidates: Writes harvest/harvest.md; refuses a finding its probe records cannot back, or a hand-written harvest.

**Usage:** `abcd lab harvest <lab-id>`

Assemble harvest/harvest.md from INTENTION.md, the findings log and the probe
records, in the lifeboat's section shape: intention, method, findings that
worked, findings still open, candidates, coverage. Each finding cites its probe
records by path; each product finding is listed as a capture candidate with
the capture line that files it, found during the lab, and flagged where no
refutation was attempted. Nothing is filed.

No claim outlives its input: a finding citing no probe record or evidence
file, or one missing or incomplete, is listed and the harvest refuses (exit 1,
nothing written). A hand-written harvest.md is never overwritten. A halted lab
is still harvested, its gate findings first.

**Example:**

```
abcd lab harvest lab-260901000000-0123abc
```

#### `abcd lab mint`

Mint a lab for one question, with a standalone snapshot at the pin: Writes only under the lab store; refuses a multi-line question or a pin naming no commit.

**Usage:** `abcd lab mint <question> [flags]`

Mint a lab for one question. The lab home is created under this repository's
lane of the lab store with a registry line; the snapshot is a standalone
clone of this repository detached at the pin (HEAD unless --pin names
another commit), made with no hook firing and with its remote cut, so
nothing done in the lab world reaches this checkout. INTENTION.md,
findings.md, corrections.md and amendments.md are scaffolded, with the
lifecycle mapped onto the home. Nothing is written into the repository.

**Flags:**

```
      --pin string   the commit the snapshot is pinned at (default HEAD)
```

**Example:**

```
abcd lab mint "does the snapshot keep the checkout's hooks from firing?"
```

#### `abcd lab preflight`

Run a lab's harness-isolation and dual-binary checks: Writes the preflight artefact, and a finding and halt on a failure; refuses an unknown lab.

**Usage:** `abcd lab preflight <lab-id>`

Run the preflight against a lab's home and write it to state/preflight.md.

Harness isolation: the lab's own HOME holds no link out of the lab; the
snapshot is a standalone clone (not a linked worktree, borrowing no object
store) descending from the pin; it has no remote; and the hooks path a
session in it would run, with the operator's global configuration in force,
resolves inside the lab.

Dual binary: bin/abcd is a regular file (never a link to an operator-level
installation) whose embedded vintage — read without running it — is the
pin, unmodified; it is the same binary the first passing preflight pinned,
since the work binary is never rebuilt; and bin/abcd-test, when present, is a
separate file.

A failed check halts the lab naming it: exit 1, the refusal recorded as a
gate finding. A preflight that passes lifts that halt.

**Example:**

```
abcd lab preflight lab-260901000000-0123abc
```

#### `abcd lab record`

Scaffold one probe record naming the artefact it observes: Writes the probe's files under state/probes/; refuses a halted lab or a probe already recorded.

**Usage:** `abcd lab record <lab-id> <probe>`

Scaffold the record one probe writes before any harvest may cite it:
input, argv, exit, stdout and stderr, empty, and record.md naming the
artefact observed (the work binary's vintage and sha256, when there is one).
The verb runs nothing: the probe's command is run by whoever runs the lab, with
its output redirected into the scaffold. A probe is recorded once and never
overwritten; a re-run is a new probe. Refused on a halted lab.

**Example:**

```
abcd lab record lab-260901000000-0123abc bare-status
```

#### `abcd lab sweep`

Sweep a lab's documents for every retracted pattern: Writes the sweep artefact, and a finding and halt on an unapplied correction; refuses an unknown lab.

**Usage:** `abcd lab sweep <lab-id>`

A retraction is a sweep, not an edit. Every correction line in corrections.md
names a literal that must be absent from the lab's own documents; the sweep
searches them all for it — the pattern, not the instance — and lists every
place it still stands. The snapshot, the lab's HOME and binaries, transcripts
and each probe's five capture files are not swept: they are the world and the
instruments, not claims; a probe's record.md is prose, and is swept. The
result is written to state/sweep.md. An unapplied correction, one too short
to mean anything, or — while any correction is recorded — a document the
sweep could not read (too large, binary, or not a regular file; each listed
by path) fails the sweep: exit 1, the lab halted and the refusal recorded as
a gate finding that names corrections by number, so it never becomes an
instance itself. A sweep that passes lifts that halt.

**Example:**

```
abcd lab sweep lab-260901000000-0123abc
```

### `abcd launch`

Preview the public launch bundle, its secret scan, and the release gates: Writes only its pre-flight report, to the local tier; refuses without --dry-run.

**Usage:** `abcd launch [flags]`

**Flags:**

```
      --baseline string   the release tag the payload parity diff measures against (default: the newest release tag)
      --deep-smoke        also run the installability smoke's deep tier: render every command, skill and agent page's help in an isolated subprocess (always on in the cut)
      --dry-run           preview the launch bundle and gates without publishing
      --fetch-baseline    read the parity baseline from the tag's published plugin archive, verified against the release's checksums.txt (a network fetch; default: a fresh render at the tag)
```

#### `abcd launch archive`

Render the release's plugin archive: Writes the archive into --out; refuses a dirty tree without --verify, and exits 1 when --verify finds it unpinned.

**Usage:** `abcd launch archive --out <dir> [--tag <vX.Y.Z>] [--verify] [--repository <owner/name>] [flags]`

Render the release's plugin archive into --out and, with --verify, prove the
committed catalog pins it (exit 1 on a mismatch).

With --verify the pin judges the working tree: a payload file that differs
from the commit changes the archive's digest, and the pin refuses it. Without
--verify nothing else judges the tree, so an uncommitted change, tracked or
untracked and outside the local tier, refuses the render (exit 2) and nothing
is written to --out.

**Flags:**

```
      --out string          existing directory to write <plugin>-plugin-v<version>.zip into
      --repository string   refuse (exit 1) unless the archive's address is this GitHub owner/name's release download for the tag
      --tag string          refuse unless the newest dated CHANGELOG version is this tag
      --verify              refuse (exit 1) unless the committed catalog pins this archive's address and digest; without it, a tree with an uncommitted change refuses (exit 2)
```

**Example:**

```
abcd launch archive --out dist
```

#### `abcd launch manifests`

Check the release manifests agree on the version, or carry none on a dev tree: Writes nothing; refuses with exit 1 on drift and exit 2 on an unreadable input.

**Usage:** `abcd launch manifests --tree public|dev [--root <dir>] [flags]`

Run the manifest lockstep check over a tree. --tree public requires the
version-location primary present as strict SemVer and every pinned secondary
to agree with it; --tree dev requires every version key absent (adr-19). The
tree is the working directory, or --root. Exit 0 consistent, 1 drift (one
line per field), 2 unreadable. Nothing is written.

**Flags:**

```
      --root string   the tree to check (default: the working directory)
      --tree string   the polarity to check: public (versions present and agreeing) or dev (versions absent)
```

**Example:**

```
abcd launch manifests --tree public
```

#### `abcd launch receipts`

Run the release job's semantic-receipt gate locally, before the merge: Writes nothing; refuses with exit 1 when the release job would refuse the receipts.

**Usage:** `abcd launch receipts`

#### `abcd launch scaffold`

Scaffold the release gate for the declared artefact kind: Writes its workflows and runbook; refuses an undeclared kind, or a hand-edited file without --confirm.

**Usage:** `abcd launch scaffold [--confirm] [--dependency-reauthor] [flags]`

**Flags:**

```
      --confirm               overwrite a hand-edited scaffolded file with the current machinery
      --dependency-reauthor   opt in to re-authoring bot-opened dependency bumps as the repository owner (seeds .abcd/config/dependency-reauthor.conf)
```

#### `abcd launch ship`

Cut a release, deriving its version and records from what shipped: Writes the CHANGELOG heading, RELEASE.md, and the archive pin; refuses a cut its gates stop.

**Usage:** `abcd launch ship [--changelog-json <file|->] [--payload-dir <dir>] [--allow-dirty] [--fetch-baseline] [flags]`

**Flags:**

```
      --allow-dirty             cut from a working tree with uncommitted changes; the pre-flight report records the override and every path it carried (waives the dirty-tree gate only — never lockstep, and never the archive pin's clean-payload refusal)
      --changelog-json string   path to the host-composed changelog JSON (or - for stdin); absent runs the deterministic emit step
      --fetch-baseline          read the parity baseline from the anchor tag's published plugin archive, verified against the release's checksums.txt (a network fetch; default: a fresh render at the tag)
      --payload-dir string      stage the versioned release payload in this directory (must be empty and outside the repository)
      --route stringArray       route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

### `abcd lint`

Check this repository against the conventions, every target but outbound: Writes nothing; refuses with exit 2 on an error finding and exit 1 on warnings alone.

**Usage:** `abcd lint [flags]`

**Flags:**

```
      --root string   repo root to lint (default: current working directory)
```

#### `abcd lint docs`

Lint the docs for change-narration, broken links, citations, and stray root markdown: Writes nothing; refuses a tree with a blocker finding.

**Usage:** `abcd lint docs [flags]`

**Flags:**

```
      --config string   path to docs-lint.json (default: <root>/.abcd/docs-lint.json)
      --release-gate    run as the release gate: a citation past its staleness threshold blocks instead of warning (release-time only)
      --root string     repo root to lint (default: current working directory)
```

#### `abcd lint identity`

Show this repository's identity block and every surface held to it: Writes nothing; refuses a repository that records no identity block.

**Usage:** `abcd lint identity`

#### `abcd lint outbound`

Judge one outbound text against the session-URL and tool-footer policy: Writes nothing; refuses a text carrying either with exit 1.

**Usage:** `abcd lint outbound [FILE] [flags]`

Judge one outbound artefact — a commit message, a pull-request body, an issue, a
comment, a release note — against abcd's outbound policy: never a live
agent-session URL, never a tool's own attribution footer.

Reads FILE, or standard input when FILE is absent or `-`. It REPORTS and REFUSES;
it never rewrites the text it was given, because the text belongs to whoever
wrote it. Exit 0 clean, 1 the artefact is refused, 2 the check could not run.

**Flags:**

```
      --label string   what the artefact is (commit-message, pr-body, issue, comment) — it names the artefact in the report (default "outbound-artefact")
      --root string    repo root supplying the scanner configuration (default: current working directory)
```

#### `abcd lint site`

Gate the built website, rendering it first when absent: Writes only inside the output directory; refuses a site failing any gate with exit 1.

**Usage:** `abcd lint site [flags]`

**Flags:**

```
      --out string   built output directory to check (rendered first if absent) (default "site")
```

### `abcd memory`

Render the memory store's status: Writes nothing; refuses outside a git checkout.

**Usage:** `abcd memory`

#### `abcd memory ask`

Query memory and synthesise a cited answer: Writes a memory page only with --file-back; refuses outside a git checkout.

**Usage:** `abcd memory ask <question> [flags]`

**Flags:**

```
      --file-back          file the synthesised answer back as a new memory page
      --page-json string   the answer page dict as JSON (file path, or - for stdin)
      --top-n int          retrieval depth (0 uses the pinned default)
```

**Example:**

```
abcd memory ask "why do record ids carry a timestamp?"
```

#### `abcd memory ingest`

Distil a local file or an https source into cited memory pages: Writes the pages; refuses a URL that is not https.

**Usage:** `abcd memory ingest <path-or-https-url> [flags]`

**Flags:**

```
      --keep-original       store the original at .abcd/memory/sources/<sha256>.<ext>
      --pages-json string   DistilledPage JSON array (file path, or - for stdin)
```

**Example:**

```
abcd memory ingest https://example.com/paper.pdf
```

#### `abcd memory lint`

Health-check the whole memory store: Writes a lint report; refuses a store with a blocker finding.

**Usage:** `abcd memory lint`

### `abcd mode`

Print or set whose answer the agent loop is waiting on: Writes the state only when setting it; refuses an unknown state or a checkout with no local tier.

**Usage:** `abcd mode [<state>]`

Print or set the waiting-on state behind the status line's badge.

Bare `abcd mode` prints the stored state: `managed` (abcd is here and nobody
is waiting), `facilitator` (the loop is parked on the facilitator, the person
at the terminal running the agents), or `product-thinker` (the loop is parked
on the product thinker, who answers on a surface of their own). An absent
store reads as `managed`.

`abcd mode <state>` sets it. Two writers share the verb: the agent runs it
when it stops for a verdict, naming whom it is addressing, and the human runs
it by hand to say which hat they wear. The state lives per checkout at
`.abcd/.work.local/mode`, so only a repository abcd manages — one that has
the local-ephemeral tier — can hold it; elsewhere the set refuses and creates
nothing. The next status-line refresh and the bare `abcd` board read the
same file.

The guard holds the agent to it: a question through the host's question
tool is refused while the state reads `managed`, and once the state names
somebody the question runs and is marked open. The next human message is
its answer, so the prompt hook resets the state to `managed` and says so
once on stderr. A state set by hand with no question open is left as it is.

Where this machine has no status surface — no `~/.abcd/statusline.json`, or
one with `disabled` set — the set form prints one line naming whose answer
is owed, once, because the verb call is the stop. Setting `managed` owes
nobody and prints nothing; with a surface installed nothing is printed at
all. With --json the notice is a field. Both forms make no network request.

Exit 2 on a refusal — an unknown state, no local tier, or no checkout —
and nothing is written on any of them.

### `abcd peers`

List the records sibling worktrees and local branches hold that this checkout does not: Writes nothing; refuses outside a git checkout.

**Usage:** `abcd peers`

List what this checkout's peers hold that it does not, before capturing, fixing or
filing anything. A peer is a linked worktree sharing this repository's git
common dir, read off its disk so an uncommitted capture is seen, or a local
branch no worktree has checked out, read from the object store.

Per peer, three kinds of row: an issue open there and absent here; an issue
open here and resolved or won't-fixed there; an intent drafted there and
absent here. A peer whose worktree directory is gone, or whose branch is
merged into the default branch (a worktree only when its record folders are
also clean), is skipped and counted. A peer git refuses to answer for, one
whose common dir is another repository's, or one whose ledger holds an id in
two status folders is named with the reason and not read; a gone or refused
worktree's branch is then read from the object store instead.

Strictly read-only: it writes nothing, takes no lock, and fetches nothing.
A worktree is named home-relative (~/...), or by its directory name when it
sits outside HOME, on every stream. Exit 0 whatever the peers hold; exit 2
outside a git checkout.

### `abcd reading`

Render the cold-reading assembler's state: Writes nothing; refuses any argument.

**Usage:** `abcd reading`

Assemble the input a cold reading is handed.

Blindness is a property of the input, not a promise the reader makes: a positive include
table names what may travel, fields are projected out of records rather than files copied
whole, and a hashed manifest records what was passed so a reader can judge contamination
rather than accept a disclosure on trust.

Bare `abcd reading` renders the assembler's state and writes nothing.

#### `abcd reading assemble`

Assemble one reading's input and its hashed manifest at one position: Writes both artefacts; refuses a target that is not HEAD or a commit sha.

**Usage:** `abcd reading assemble --position <position> --target <HEAD|sha> [flags]`

Walk the repository under the include table at one reading position and write two
artefacts: the assembled input, which carries no repository path, and the manifest,
which maps every passed item back to its path, its field and its hash.

The invocation is a position and a target state, and nothing else. --position takes
one of four closed tokens; --target takes HEAD or a hexadecimal commit sha of 7 to 40
digits, because a branch or a tag moves and the manifest's re-runnability rests on a
reference that cannot. Both are required.

What the reading is handed comes from the committed preset entry for the position, in
.abcd/config/reading-presets.json, applied with no operand. Changing it is a commit to
that file, reviewed and inside the dirty gate; the manifest records the entry applied
and its hash, so a run is reproducible from the commit it names.

**Flags:**

```
      --dry-run           write nothing; with --out the two artefacts still land in that directory
      --out string        an empty or absent directory the assembled input and the manifest are written to,
                          for inspection: reading ingest finds a run only in the local-tier run directory,
                          so a run written here cannot be ingested (default: the local-tier run directory)
      --position string   the reading position: widening, entailment, comparative, detection
                          (comparative derives its candidate set from the record: the one committed
                          widening run at the target whose items carry no disposition and no
                          admission — at the target, or at an ancestor of it across which only the
                          readings store and the issue ledger changed, so a run's own records can be
                          committed between its ingest and this assembly. None, or more than one,
                          refuses and lists the runs)
      --target string     the commit the assembly describes: HEAD, or a hexadecimal sha of 7 to 40 digits
```

**Example:**

```
abcd reading assemble --position widening --target HEAD
```

#### `abcd reading ingest`

Validate the JSON one cold reading returned: Writes its reading records; refuses output the position's licence does not allow.

**Usage:** `abcd reading ingest --reading-json <path> [flags]`

Validate the JSON a cold reading returned and write its reading records.

The verb checks what the reading was LICENSED to produce, not only what it saw: the
supply regime is read from the position's definition and compared with the output's own
claim, and an item carrying a reserved name as one of its own fields is refused with the
licence stated. The reserved-name table is read at the run's own regime, one row per
regime, and the generative regime has no row: no name is reserved at the generative
position.

Item identifiers are minted here. The payload carries none, so a supplied one is refused
as an unknown field. A refusal becomes DURABLE once the run's identity is proven — the run
id resolving to a parked manifest whose content hash matches — and from there a list-level
refusal writes refusal.json under the run's directory; before that point nothing durable is
written anywhere. No OTHER run's durable state is touched until the whole payload validates:
a refusal after the run is proven writes its refusal record and nothing else, and the one
delete it makes is on its OWN run id — the records of an earlier attempt at it that never
committed. The reading records land as one batch and the run metadata is written last as the
commit marker: a run without one never happened.

An ingest interrupted before that marker leaves an orphaned stage, and every invocation names
it. Only the next one whose payload validates sweeps it: where the run reached no commit
marker the sweep ROLLS THAT RUN'S READING RECORDS OUT OF THE COMMITTED LEDGER, because the
run never happened; where the marker is there the run stands and only the stage goes. A
refused run reports the orphans it left in place, and the ids a sweep removed are reported as
rolled_back_records on every exit, including a failing one.

Every stored finding is matched against the record as a capture is: its pattern and body are
compared with the open and resolved issues, the intents and every earlier reading item, never
with another item of the same run, and a likely repeat is written onto the reading record as a
duplicates: or refines: link and shown, printed and as matches in --json.

**Flags:**

```
      --reading-json string   path to the JSON the cold reading returned
      --route stringArray     route one agent for this run: <agent>=<tier>[@<connection>][?k=v,...], tier one of local | economy | frontier | host-decides (one per agent this invocation dispatches, and each invocation dispatches one; wins over every accepted routing table for this run alone, and the receipt records it verbatim)
```

**Example:**

```
abcd reading ingest --reading-json reading.json
```

### `abcd report`

File a defect report or an enhancement proposal about abcd: Writes it into your account's inbox; refuses a malformed field or a filesystem path.

**Usage:** `abcd report [<file>|-] [flags]`

File a written account about abcd itself, from a repository abcd manages, into
the inbox in the user account's machine store (`~/.abcd/inbox/`). Nothing is
written into this repository or into abcd's, and nothing becomes a record until
a person or a session runs `abcd inbox promote`.

`abcd report --template` prints the skeleton: a block of fields between `---`
lines (template version, kind, severity, category, title, the abcd version and
surface in play, an optional remedy and evidence pointers) and the prose below
it. `abcd report <file>` validates a filled report and files it; `-` reads it
from stdin. Bare `abcd report` opens the skeleton in $VISUAL or $EDITOR when
the session is a terminal, and files what is saved.

The report is held to the template: a missing or malformed field is refused
naming the field, a report over 32 KiB or carrying a control byte is refused,
and a field naming a filesystem path is refused, because a report points at
records, commits and URLs, never at a location on a machine. abcd names the
file from the time and this repository's root-commit key; the verb prints the
report's id and where it landed.

Exit 2 on a refusal, with nothing filed. Exit 1 when filing fails (the inbox
cannot be created, every id drawn this second is taken, the write fails), with
nothing filed. After the editor ran, both name where what was written is kept.

**Flags:**

```
      --template   print the report skeleton to fill (writes nothing)
```

### `abcd rules`

Render the active rule set, or the one domain named: Writes nothing; refuses an unknown domain.

**Usage:** `abcd rules [domain]`

Render the rule set the modular-rules loader injects: the bundled default
domains, overridden by this machine's ~/.abcd/rules.json and then by this repo's
.abcd/rules.json, each layer per field, so the repo wins a field both set.
Either file may be absent. Bare, it renders every active domain; a positional
DOMAIN (case-insensitive) renders that one domain regardless of its state or the
kill switch, so a dormant domain is still inspectable.

Every domain says which layer it came from. A domain an override names — its
rules replaced, its state changed, or a custom domain declared — renders as
"## NAME (user override)" or "## NAME (repo override)" here, in the injected
block and in the hook's diagnostic, and carries "source": "user" or "repo" in
--json; the last layer to name a domain labels it. An untouched bundled domain
renders bare and carries "source": "bundled".

A list an override sets replaces the bundled one, so an override can hold back
an entry abcd ships. For the guardrail domains (COMMITTING, LOAD, PII, SHELL),
every bundled recall keyword, alias or rule that an override's list leaves out
is named on stderr, with the file that set the list, here and on every hook
prompt. To keep an entry, restate it in the list, or leave the field out to
inherit the bundled list.

SHELL is generated from the bundled shell-hazard registry that "abcd guard"
enforces: one rule per registry entry, naming the command, why it is dangerous
and what to run instead, recalled by the commands the registry names. It
teaches before shell work what the guard refuses at the moment a command runs.
Read-only.

### `abcd scribe`

Assemble a ledger scribe's context and ingest what it transcribed: Writes nothing bare; refuses an unknown sub-verb.

**Usage:** `abcd scribe`

Build the ledger scribe's context and ingest what the scribe returns.

The scribe transcribes a reading run's records and the researcher's dispositions into the
ledger's declared shapes, and authors nothing. Its context is the reading assembler's exact
inverse: ledger content only, drawn from the issue ledger's own directories, and the
researcher's supplied text. `assemble` builds it with a manifest of every path passed;
`ingest` validates the scribe's output and refuses anything the scribe authored.

#### `abcd scribe assemble`

Build a scribe session's context from the ledger and supplied dispositions: Writes it and a hashed manifest; refuses an uningested run or a symlinked ledger.

**Usage:** `abcd scribe assemble --run <rdg-N> --dispositions <path> [flags]`

Build the context one scribe session is handed, for one ingested reading run.

The context is positive inclusion at directory grain: the issue ledger's own directories
(its reading records, dispositions, admissions, surprises and reframes, and its three status
directories), derived from the ledger's directory list, and the researcher's dispositions
text read whole. Nothing else is walked, and an item outside that list is refused whatever
route it arrived by. The run must be ingested: its records come from the store, never from
a raw reading output handed over again.

The context and a manifest naming every path passed, by hash, are parked in the local tier
(or under --out, which may not be a directory a reading's include table reaches). Nothing in
the durable record is touched. Both carry the scribe's per-run context stamp.

**Flags:**

```
      --dispositions string   the researcher's dispositions text, read whole and carried verbatim
      --dry-run               write nothing; with --out the two artefacts still land in that directory
      --out string            an empty or absent directory the context and the manifest are written to
                              (default: the local-tier scribe run directory)
      --run string            the ingested reading run the session transcribes for (rdg-N)
```

**Example:**

```
abcd scribe assemble --run rdg-2609010000000001 --dispositions dispositions.md
```

#### `abcd scribe ingest`

Validate a scribe's output against the supplied dispositions: Writes what it transcribed through the capture verbs; refuses anything the scribe authored.

**Usage:** `abcd scribe ingest --scribe-json <path> --dispositions <path> [flags]`

Validate the JSON a scribe session returned and write its records through the capture verbs.

The context the session was handed is proven first: it must hash to its parked manifest, and
the output must cite that hash. The pair is parked where a scribe session could rewrite it, so
--dispositions names the researcher's own text again, the file assemble was handed: the
manifest's supplied hash and the context's supplied copy must both equal it, and every check
below reads it. Then the output is refused if the scribe authored anything —
a field outside the declared shapes, an item the supplied dispositions never name, a state or an
admission the item's own line of the supplied text does not carry, or a ground, exit condition
or surprise that does not stand verbatim in the supplied text once whitespace is folded — or if
it passes over an unanswered item of the run in silence. Nothing is written until all of that
holds.

Dispositions, admissions and surprises are then written in that order through the capture verbs,
which apply their own redaction and refusals, the ordering gate included; the first refusal stops
the ingest and names what landed before it. Fidelity flags and refusals are reported and never
written. Once every write has landed, and when at least one record did, the manifest is promoted
beside the run, write-once; an ingest that lands no record leaves the run open.

**Flags:**

```
      --context string        the context the session was handed, when assemble wrote it under --out
                              (default: the local-tier scribe run directory of the output's run)
      --dispositions string   the researcher's dispositions text, the file assemble was handed
      --scribe-json string    path to the JSON the scribe session returned
```

**Example:**

```
abcd scribe ingest --scribe-json scribe.json --dispositions dispositions.md
```

### `abcd site`

Report what the website declares and what was built: Writes nothing; refuses any argument.

**Usage:** `abcd site [flags]`

**Flags:**

```
      --out string   output directory to report on (default "site")
```

#### `abcd site build`

Render the website into the output directory: Writes only inside that directory; refuses a non-empty directory it did not write.

**Usage:** `abcd site build [flags]`

**Flags:**

```
      --commit string    commit for the footer and the build stamp (default: git HEAD)
      --date string      date for the build stamp (default: the newest release's date)
      --out string       directory to render into (default "site")
      --preview          stamp the build as unreleased at this commit, for a preview deployment of an untagged tree
      --version string   version for the footer and the build stamp (default: the newest dated CHANGELOG heading)
```

#### `abcd site setup`

Take the website from this checkout to a live address: Writes its files, and the forge and host changes once confirmed; refuses a folder abcd does not manage.

**Usage:** `abcd site setup [flags]`

**Flags:**

```
      --confirm         replace a workflow or host configuration that differs from what setup writes
      --domain string   custom domain to route to the host when the composition names none
      --name string     host name when the composition names none (default: the repository's name)
      --yes             confirm the forge and host changes without being asked; without it an unanswered run declines them
```

### `abcd source`

Render the sources corpus and its ledgers, read-only: Writes nothing; refuses without a corpus, exit 3, naming `abcd source init`.

**Usage:** `abcd source`

The personal sources corpus: documents you may consult, a CSL-JSON bibliography, and one
append-only influence ledger per repository, in a local-only git repository with no
remote (~/.abcd/sources by default; --corpus names another). The folder a source sits
in — confidential/<key>/ or public/<key>/ — is its classification.

Consult freely, cite deliberately: confidential entries are projected into this
repository's untracked private banlist (sync-banlist), which the committed pre-commit
guard refreshes and enforces; cite-check clears text before it leaves the machine; and
a ledger line becomes a public citation only when the source permits it AND a person
flips the line (adr-41). No output names a confidential source except by key.

Bare `abcd source` is read-only. Exit 3 when there is no corpus, on every verb but init.

**Flags:**

```
      --corpus string   the corpus directory (absolute; default ~/.abcd/sources)
```

#### `abcd source add`

Register a source under its class folder, with its entry and text: Writes the corpus and commits it; refuses without one of --confidential or --public.

**Usage:** `abcd source add [document] [flags]`

Register a source: write its CSL-JSON entry (with the custom block), store the document
as original.<ext> and its extracted text as text.md under confidential/<key>/ or
public/<key>/, and commit the corpus. The class is declared here, once: exactly one
of --confidential or --public is required. abcd converts nothing and fetches nothing:
a Markdown or text document is its own text, any other needs --text, and a URL
alone registers a metadata stub.

A confidential entry's title, aliases and (under --ban-authors) authors become banned
phrases, so each must hold at least three letters or digits, and its key must not
contain any of them — the key is what every refusal and scan prints. Pass those
strings with --meta FILE (or --meta - on stdin) to keep them out of argv and shell
history.

**Flags:**

```
      --alias stringArray      another identifying name for a confidential source (repeatable)
      --author stringArray     an author, "Family, Given" or a literal name (repeatable)
      --ban-authors            also ban the authors' names (a confidential source whose authorship is itself identifying)
      --confidential           file the source under confidential/ (exclusive with --public)
      --key string             the source key: lowercase, opaque for a confidential source (e.g. conf2026a)
      --keywords stringArray   retrieval keywords, comma-separated (repeatable)
      --meta string            a JSON file (or - for stdin) with "title", "aliases", "author" and "keywords"
      --permission string      permission_status: citable | no-public-citation | internal-never-cite | ai-generated-never-cite | ask-author (default by class)
      --public                 file the source under public/ (exclusive with --confidential)
      --text string            the extracted text of a non-text document
      --title string           the exact title (for a confidential source prefer --meta)
      --type string            the CSL item type (default document)
      --url string             the canonical URL (recorded, never fetched)
      --venue string           the container title (journal, site, publisher)
      --year int               the year of issue
```

#### `abcd source cite-check`

Scan text for confidential sources and report each hit by key only: Writes nothing; refuses without a corpus, and exits 1 on a hit.

**Usage:** `abcd source cite-check <file|->`

Scan a file, or stdin with -, for every confidential source's title, aliases and
opted-in authors, through the private banlist's matcher — the engine the pre-commit
guard runs. Offenders are reported by key, field, line and byte offset, never by the
text matched, so the report is safe to relay. The offset counts bytes from the start
of the whole text, not from the start of the line, to the start of the matched span,
which can be the one byte before the phrase that bounds it. Exit 1 when anything is found.

**Example:**

```
abcd source cite-check draft.md
```

#### `abcd source declassify`

Move a published confidential source to public/ in one visible commit: Writes the corpus and commits it; refuses a key that is not confidential.

**Usage:** `abcd source declassify <key> [flags]`

Declassify a confidential source once it is published: `git mv` its folder from
confidential/ to public/ and set the entry's confidential flag and permission_status
(citable unless --permission says otherwise), in one corpus commit. The next
sync-banlist drops its strings, and its ledger lines become flippable.

**Flags:**

```
      --permission string   permission_status after the move: citable | no-public-citation | internal-never-cite | ai-generated-never-cite | ask-author (default citable)
```

**Example:**

```
abcd source declassify example-paper-2026
```

#### `abcd source init`

Create an empty sources corpus, a git repository with no remote: Writes the corpus in one commit; refuses an existing corpus or one inside another working tree.

**Usage:** `abcd source init`

Create the corpus at its location (0700): a git repository with no remote, an empty
sources.json and a README, in one commit. Refuses an existing corpus, a non-empty
directory, and a location inside another repository's working tree.

#### `abcd source ledger`

Append an influence line to this repository's ledger, or flip one to cited: Writes one ledger line; refuses a flip for a source that is not public and citable.

**Usage:** `abcd source ledger [flags]`

Append one influence record — {ts, repo, decision_ref, claim, source_key, locator,
influence, cited_publicly: false} — to this repository's ledger in the corpus, and
commit it. The ledger is append-only: a correction is a new line (--corrects N).

--flip N is the person's act of citing line N publicly. It checks the source first
(adr-41 gate 1: the folder is public/ and permission_status is citable), refuses
naming the failing gate, and on success appends a NEW line with cited_publicly true.
An agent never runs it. --list prints the ledger, numbered.

The repository is named by its root commit's first twelve hex digits unless --repo
names it.

**Flags:**

```
      --claim string          what was decided or claimed
      --corrects int          the line this record corrects
      --decision string       the decision influenced: a DECISIONS.md date, an ADR or intent id, or free text
      --flip int              cite line N publicly (the person's act; checks the source's permission first)
      --influence string      the influence: supports | contradicts | method | background
      --list                  print the ledger, numbered (read-only)
      --locator string        where in the source (pp., §)
      --repo string           the ledger's repository handle (default: this checkout's root commit, 12 hex digits)
      --source string         the source key
      --used-in stringArray   a repository-relative path the influence landed in (repeatable)
```

#### `abcd source sync-banlist`

Project confidential titles and aliases into the private banlist: Writes the store's generated block; refuses a corpus whose folders and entries disagree.

**Usage:** `abcd source sync-banlist [flags]`

Regenerate the corpus's block in this repository's untracked private banlist
(.abcd/.work.local/private-names.txt, the banlist verb's private layer): every
confidential source's title and aliases, and its authors under ban_authors, as
whitespace-flexible, case-insensitive phrases. Lines outside the block survive.
A corpus whose folders and entries disagree is refused and nothing is written.

--refresh is the pre-commit guard's mode: it updates a private store that already
exists and declares the keyed format, and never creates one. With no corpus, no store
or a legacy store (migrate it with `abcd banlist migrate`) it says so on one line and
exits 0.

**Flags:**

```
      --refresh   the guard's mode: update an existing store only; an absent corpus or store is a one-line notice and exit 0
```

### `abcd spec`

Render the spec store's status: Writes nothing; refuses outside a git checkout.

**Usage:** `abcd spec`

#### `abcd spec close`

Close a spec, and ship its intent when no open spec names it: Writes the moves to closed/ and shipped/; refuses to ship an intent with no impact.

**Usage:** `abcd spec close <spc-N> [flags]`

Moves the spec to closed/ and, when no open spec still names its intent, moves the intent to shipped/.

The close that ships an intent also makes its fidelity review owed: it mints an OWED receipt (rcp-…), parks an `<!-- abcd-review: OWED receipt=rcp-… -->` marker in the intent's Audit Notes, and writes the review request to `.abcd/.work.local/reviews/<rcp>.request.md`, the input `abcd intent audit ingest` answers. A failed emit is a warning on stderr; the intent ships regardless.

**Flags:**

```
      --impact string            product impact to stamp on an intent that declares none: additive|breaking|fix (an intent may not be internal); accepted only at the close that ships the intent
      --production-mode string   how this record's text was produced: hand-written|dictated-and-formatted|scribe-transcribed (default: the repo's declared mode, else hand-written)
      --remainder string         kebab-case slug of a follow-on spec to mint for what this spec did not deliver, attached to the same intent (which then stays planned); it carries the steps not marked landed
```

**Example:**

```
abcd spec close spc-2609010000000001
```

### `abcd statusline`

Render abcd's status-line row from the host's payload on stdin: Writes nothing; never refuses.

**Usage:** `abcd statusline`

Render abcd's row for the host harness's status line.

The harness runs this on every status refresh, with its JSON status
payload on stdin, and shows what it prints. In a checkout abcd manages the
row is abcd's own: the presence badge first — `abcd-managed`, `waiting on
the technical facilitator` or `waiting on the product thinker`, from the state
`abcd mode` stores, its colour ending at the badge — then the
repository name, the branch, the model, the context percentage, the
five-hour and seven-day usage percentages, and the record's counts of
intents not yet shipped and open issues. Each element after the badge is
switchable in `~/.abcd/statusline.json`; a payload field the harness did not
supply drops its element with no placeholder.

Outside a managed checkout, or with `disabled` set in the user-level
setting, it runs the status command recorded there at install time with the
same stdin and passes its output and exit code through unchanged, so the
user's own line is untouched everywhere abcd does not manage. With none
recorded it prints nothing and exits 0.

The checkout is resolved from the payload's `cwd` (falling back to the
working directory). Empty stdin is an empty payload. Nothing here prompts,
reads a terminal, or touches the network. With --json the row is emitted as
its ordered elements, each with a key, a rendered and a plain form.

### `abcd update`

Swap the PATH-installed binary for a verified release, or with --check only compare: Writes the swapped binary; refuses a binary it cannot prove is abcd's.

**Usage:** `abcd update [tag] [flags]`

Fetches the named release (or resolves the latest, naming it before acting),
verifies the platform binary against the same release's checksums.txt, and
swaps the PATH-installed copy atomically. The verb is the only ask: abcd
never checks for or applies updates on its own (adr-38). A plugin-root
binary, the dev shim, and package-manager installs are refused with the
command that owns them. The file being replaced must be provably abcd's:
the binary running the command, an install ~/.abcd/path-entry records, or
a digest a published release still names. Anything else is refused with a
remedy that reinstalls over it — never one that deletes it.

With --check it only asks: it fetches the latest release's tag once, says
whether this binary is behind and which command takes the update for this
install's shape, and swaps nothing.

**Flags:**

```
      --check   fetch the latest release once and compare it with this binary, swapping nothing (the only network touch besides the update itself; abcd never fetches implicitly — adr-38); names its source and the command that takes the update
      --yes     skip the TTY confirmation of a freshly resolved tag
```
