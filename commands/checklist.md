---
name: checklist
description: "List this working copy's checklist, every item and field: Writes nothing; refuses outside a git checkout."
argument-hint: "[add|set|show|render]"
block: agents
user-invocable: false
---

# `/abcd:checklist`

The working copy's checklist: what this session is working on, what it waits
on and what it has promised. It lives in `.abcd/.work.local/checklist.json`,
one file per working copy, so a resumed or a fresh session in the same copy
reads the same list, and it survives compaction and a crash. Keep it current
as you work, through these verbs only: the file is checked when it is read,
and a hand edit the reader cannot vouch for is passed over.

## List it

Bare invocation is read-only. Run:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" checklist --json
```

The payload carries `source` (`checklist`, `last-good` or `nothing`),
`reason` (one sentence saying which file was read and why) and `checklist`,
absent when there is none. The reader validates and scans checklist.json;
when it fails, it shows the last copy a checked write produced, and when both
fail it shows nothing. Tell the user the `reason` whenever `source` is not
`checklist`.

## Add an item

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" checklist add "Regenerate the CLI reference" --json
```

The verb mints the item's id from the title, a short slug with `-2`, `-3`
and so on when it is taken, and prints it as `id`. The id never changes
after: `set` and `show` take it. The item is `working`, or `waiting` when
`--waits-on` names what it waits on: a record id, `pr:<number>`,
`person:<role>` or `slot:<kind>`. `--next-look` is when to look again, an
RFC 3339 timestamp, and `--promise "<trigger> => <act>"` records what you
promise to do when something happens, split at the first `=>`. The title is
one line of at most 200 characters.

## Change an item's state

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" checklist set regenerate-the-cli-reference --state done --json
```

The states are `working`, `waiting`, `done` and `dropped`.

## Show one item

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" checklist show regenerate-the-cli-reference --json
```

## Render it

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" checklist render
```

Prints the checklist as one block of text: the open items
first, then the done and dropped ones, one line each. With `--json` the text
is the `text` field.

## Refusals

Every write validates the whole checklist against its schema and scans every
field for secrets before anything is written. A refusal names the field and
the rule or the pattern kind, never the value, exits 2, and leaves both files
as they were. Never paste a credential into the checklist: an item that needs
one names its kind, scope and expiry only. An unknown id, an unknown state, a
`--promise` without `=>`, no local tier and no checkout are refused the same
way. Relay the refusal to the user as it is.

A write starts from what the reader returns: when checklist.json fails its
checks it builds on the last good copy, and when both fail it starts a new
list; either way it says so (on stderr, or as `notice` with `--json`). Relay
that line too.

`updated_by` records the host's session id where the host exports one, and
otherwise the name the away record would give the writer. No verb makes a
network request.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
