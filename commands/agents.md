---
name: agents
description: "List the agents abcd started for this session and add them up: Writes nothing; refuses when no session can be named and no scope is given."
argument-hint: "[--run <id> | --session <id> | --all]"
block: people
---

# `/abcd:agents`

List the agents abcd started and add them up, from the launch ledger on this
computer (`~/.abcd.noindex/agents/`). The verb reads only: it writes nothing and
changes no entry.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" agents --json              # this session's launches
"${CLAUDE_PLUGIN_ROOT}/abcd" agents --run <id> --json   # one run's, whoever drove it
"${CLAUDE_PLUGIN_ROOT}/abcd" agents --session <id> --json
"${CLAUDE_PLUGIN_ROOT}/abcd" agents --all --json
```

Bare, it lists the session the command runs under: the one the host exports, or
one whose host process the command descends from. Under neither, as in a plain
terminal, it refuses (exit 2) and names the flags. Two scopes at once are refused
the same way.

## What an entry says

One launch per entry: the session or run that asked for it, the role and the
work, the runner, the billing route, the model that answered, the tokens by kind,
the cost, the API time, the wall time and how it ended. A field the harness did
not report is absent, never a guess.

The outcome is `running`, `finished`, `failed` (with its reason), `stopped` (killed
at its time bound or with its step, carrying what its transcript shows it spent),
`handed-to-host` (a placeholder with no figures, since the host counts its own
agent) or `lost`, which a reader judges: an entry marked running whose process is
gone.

## Reading the totals

- **Money is counted for a named key alone.** `spend_usd_by_key` sums the launches
  billed to each key. A launch whose stream shows a subscription's rate-limit
  windows, and one abcd can place in neither, is summed in `estimate_usd`, which is
  what the work would have cost and no money spent. Say so when you relay it.
- **Every cost is the harness's own estimate** at its price table, and the
  provider's invoice is the final figure; `cost_note` says this and goes with any
  figure you relay.
- **A token total states its basis.** `tokens` counts every count the harness
  reports (`token_basis`), almost all of which are usually cache reads.
- `repeats` names every await two or more launches ran for: the first place a role
  paid for twice shows. Name it to the person.
- `unreadable` names each file the reader could not take as an entry, and `newer`
  each entry a later abcd wrote, read as far as this one can. Relay both: a total
  that passes over an entry is not the whole figure.
- `liveness_unread` says the process table could not be read, so a running entry
  stands as written.

## The store

One file per launch, each holding one line of JSON, written by the process that
launched and replaced when the launch ends. It lives under abcd's home on the
machine that launched, so removing a worktree or a checkout deletes no entry, and
another machine keeps a ledger of its own.

The next launch prunes the oldest whole files past 30 days and 20,000 entries,
and never one inside 32 days, the longest calendar month a key's budget counts
plus a day; the shipped 30 days therefore holds 32. `ledger.retention_days` and
`ledger.retention_count` in `~/.abcd.noindex/config.json` set either figure, up to
36,500 days and 1,000,000 entries, and a days figure below the floor is read as the
floor and named as such. The store holds every repository's launches, so the same
keys in a repository's `.abcd/config.json` may lengthen what this computer keeps and
never shorten it: a shorter figure is named and ignored.

An entry that cannot be written refuses the launch, naming the store: no launch
escapes the record.
