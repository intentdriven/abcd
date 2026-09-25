---
name: report
description: Tell abcd about a defect or propose an enhancement from a repository it manages — fill the abcd-issued template and file it into the inbox in the user account, by invoking the abcd binary. `--template` writes nothing; filing writes only to ~/.abcd/inbox/, never to either repository.
argument-hint: "[--template | <file> | -]"
block: agents
---

# `/abcd:report` — report back to abcd

Use this when working in a repository abcd manages turns up something about
**abcd itself**: a verb that misbehaved, a refusal that was wrong, a page that
said one thing while the binary did another, or an enhancement worth proposing.
The report waits in the user account's inbox, abcd names it at its next start,
and nothing becomes a record until a person or a session promotes it with
`/abcd:inbox`. Findings about the repository you are working in go to
`/abcd:capture` in that repository instead.

## Get the template

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" report --template
```

It prints the skeleton and writes nothing: a block between `---` lines —
`kind` (`defect` or `enhancement`), `severity`, `category`, `title`, the
`abcd_version` already filled in, the `surface` in play, an optional `remedy`,
and `evidence` pointers — with the prose below it.

## Fill it and file it

Write the account in the prose, and keep every block value on one line.
Evidence pointers are record ids, commit SHAs and URLs; **never put a path on
this machine anywhere in the report**. A field naming an absolute,
home-relative, `$HOME` or `..` path is refused; that check reads the fields
only and is best effort, so keep paths out of the prose too. Invisible or
direction-changing characters (zero-width spaces, bidi controls) are refused
wherever they appear. Keep it short (under 32 KiB): a run's
whole account belongs in a document the report points at.

File it through stdin, so no file is left in either repository:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" report - --json <<'REPORT'
<the filled template>
REPORT
```

or name a file you wrote outside the repository's tracked tree:
`"${CLAUDE_PLUGIN_ROOT}/abcd" report <file> --json`. A person at a terminal can
run bare `abcd report`, which opens the skeleton in `$VISUAL` or `$EDITOR`.

On success, tell the user the report's `id` and its `path` (under
`~/.abcd/inbox/`). A refusal exits 2, names the field, and files nothing: fix
that field and file again. A report whose `schema_version` this abcd does not
know is refused naming the version.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
