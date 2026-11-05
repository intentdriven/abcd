# `/abcd:agents` — List and Add Up the Agents abcd Started

`/abcd:agents` reads the launch ledger (itd-2611031342011261,
spc-2611041701500441): one entry per agent abcd starts, under abcd's home on the
machine that launched. It is a person's verb, listed in the records group, and
it runs in the process, touching no network.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|

Bare, the verb lists the launches of the session the command runs under; a flag
names one run, another session or every launch instead. With no session to name
and no scope given it refuses at exit 2, as it does for two scopes at once.

## The record

`runner.RecordLaunch` is the one way an entry is written. It writes the entry
marked running, runs the launch it is handed, and completes the entry from what
the launch returned; handed no launch, it writes the placeholder for a role
handed to the host. The runner's dispatcher wraps every adapter run in it, so a
build's and a drain's lanes, the drain's judge and the planning interviewer are
covered, and a launcher outside the dispatcher calls the same function. A launch
it cannot record is not started.

The store is `~/.abcd.noindex/agents/`, one file per launch holding one line of
JSON. The run state names nothing of it, and its schema version does not move.
A reader takes what it knows, names an entry above its own version and any file
it cannot read, and never writes; running is judged on the process, and an entry
whose process is gone reads as lost. The billing route is what abcd handed or
what the stream shows, never an account it looked up: a named key is the key
route, a stream carrying rate-limit windows is the subscription's, and neither is
unknown. Only a key's cost counts as spend, and no key's value is ever read or
written.

Retention is pruned by the next writer: 30 days and 20,000 entries unless
`ledger.retention_days` or `ledger.retention_count` says otherwise, and never
inside 32 days, the longest calendar month a key's budget counts plus a day. The
store holds every repository's launches and each launch prunes it by the figures
it read, so only the machine's figure binds every launch: a repository's figure
holds for that repository's launches, and only where it is longer, and another
repository's launch still prunes by its own. The next writer also clears a
temporary file a crash left more than a day ago.

An entry holds no prompt, no answer and no credential. A key is named by its
credential-store name, and a launch handed a name shaped like a value is refused;
a failure's reason has every secret span sealed, and a validator's refusal, which
quotes the answer, is recorded only as a refusal.

The status line's row and an unattended run's summary read this ledger once they
ship; they are the spec's remaining steps.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd agents`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--all` | bool |
| `--run` | string |
| `--session` | string |

<!-- surface-appendix:end -->
