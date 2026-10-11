---
name: memory
description: "Count the memory notes in both layers and name any layer it cannot read: Writes nothing; refuses an unknown sub-verb."
argument-hint: "[add|list|remove <mem-id>]"
block: agents
user-invocable: false
---

# `/abcd:memory`

Memory notes are short lessons you keep so that a later session does not have
to learn them again: a flag the tests need, a preference the person stated, a
trap in the tooling. You write a note without asking. Each one is stamped with
who wrote it, when, and in which session. Personal notes live on this machine,
in `~/.abcd.noindex/memory.json`. Notes a person has promoted into the project
live in its `.abcd/memory.json`, and this command never writes that file.

## See what is kept

Bare invocation is read-only:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory --json
```

It counts the personal notes tied to this repository, those recalled in every
repository, and the notes in the project's file. If a layer cannot be read, it
names the file and the reason. Tell the user about any unreadable layer.

To read the notes themselves, with each one's body, recall words, stamps and
reach:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory list --json
```

## Write a note

Write a note when you have learned something you would otherwise have to
learn again next session. Keep to one lesson per note. Give it a short title
and a body of a few sentences. Choose recall words that a later prompt would
actually contain (`test`, `release`, `migration`): the note comes back when a
prompt matches one of them. Write the body in your own words, and pass it on
stdin:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory add --json \
  --title "The tests need the race flag" \
  --recall test --recall race \
  --by "<Vendor>:<model-version>" \
  --session "${CLAUDE_SESSION_ID}" <<'NOTE'
Run go test -race ./internal/... before saying the tests pass; the plain run
misses the data race in the hook tests.
NOTE
```

- `--by` is your own vendor and model, the same value you would write in an
  `Assisted-by:` trailer.
- `--session` is filled in by the host with its own id for this session. Pass
  it as written above and never guess it. If the host leaves it empty, the
  note records its session as unknown, and the command says so on that write.
- A note is tied to this repository. Add `--everywhere` only for a lesson that
  holds in every project. Such a note has been checked against this
  checkout's private names only, and the command says so: a name another
  project bans is yours to keep out of it.
- Never put a secret, a private name or a network address in a note. The
  command refuses one, names the field and the kind of thing it found, and
  writes nothing. Rewrite the note without it; never try to get past the
  refusal.

Relay the printed `id` and every line of `notices` to the user.

## Move a lesson from the host's own memory

When the person points at a lesson in the host's own memory files, or pastes
one, and only then, read the passage they named with your own tools and file
it with `--moved-from-host`, the text on stdin as above. The command takes no
path, so it never opens those files itself. Never scan the host's memory files
for candidates, and never edit or delete the original: removing it there is
the person's choice, in the host.

## Remove a note

Run this only when the person asks:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" memory remove <mem-id> --json
```

It removes any personal note, whoever wrote it. A note in the project's file
is refused: it is removed by editing `.abcd/memory.json` in a reviewed change.
Until the project's review on the dashboard ships, `list` and `remove` are how
a person reviews the notes.

Every refusal exits 2 and writes nothing. A machine file the read refuses,
one that fails a trust check or does not parse, exits 1 and is left as it was.
The command makes no network request.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
